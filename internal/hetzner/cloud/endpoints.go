/*
 * Endpoints - functions for handling and transforming endpoints.
 *
 * Copyright 2026 Marco Confalonieri.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package hetznercloud

import (
	"fmt"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/idna"
	"sigs.k8s.io/external-dns/endpoint"
)

// fromHetznerHostname converts Hetzner DNS hostname format back to FQDN for ExternalDNS.
// This is the inverse of adjustCNAMETarget() and adjustMXTarget() in change_processors.go.
// Hetzner uses zone-relative hostnames: "mail" for local, "external.com." for external.
// ExternalDNS works WITHOUT trailing dot internally, so we return names without it.
//
// Key insight: Hetzner convention is that EXTERNAL hostnames have trailing dot,
// while LOCAL hostnames (within the zone) do NOT have trailing dot.
//
// References:
//   - Hetzner DNS docs: "When there is no period at the end, the zone itself is appended automatically"
//     https://docs.hetzner.com/dns-console/dns/record-types/mx-record/
//   - DNS trailing dot convention: https://docs.dnscontrol.org/language-reference/why-the-dot
//
// Examples (zone = "alpha.com"):
//
//	"@"              → "alpha.com"       (apex record)
//	"mail"           → "mail.alpha.com"  (local subdomain)
//	"a.b"            → "a.b.alpha.com"   (deep local subdomain)
//	"external.com."  → "external.com"    (external, has trailing dot → strip it)
//	"mail.beta.com." → "mail.beta.com"   (external, has trailing dot → strip it)
func fromHetznerHostname(zone string, host string) string {
	// Handle apex record
	if host == "@" {
		return zone
	}

	// Hetzner convention: trailing dot means EXTERNAL hostname (outside of zone)
	// No trailing dot means LOCAL hostname (within zone)
	if strings.HasSuffix(host, ".") {
		// External hostname - just strip the trailing dot
		return strings.TrimSuffix(host, ".")
	}

	// Local hostname (no trailing dot) - append zone
	return host + "." + zone
}

// makeEndpointName makes a endpoint name that conforms to Hetzner DNS
// requirements. It converts an FQDN to a zone-relative name.
func makeEndpointName(domain, entryName string) string {
	// Trim the domain off the name if present.
	adjustedName := strings.TrimSuffix(entryName, "."+domain)

	// Record at the root should be defined as @ instead of the full domain name.
	if adjustedName == domain {
		adjustedName = "@"
	}

	return adjustedName
}

// makeEndpointTarget makes a endpoint target that conforms to Hetzner DNS
// requirements.
func makeEndpointTarget(entryTarget string) (string, error) {
	// Trim the trailing dot
	trimmedTarget := strings.TrimSuffix(entryTarget, ".")
	// non-ASCII records are now supported?
	adjustedTarget, err := idna.ToASCII(trimmedTarget)
	if err != nil {
		return "", err
	}

	return adjustedTarget, nil
}

// extractEndpointTargets extracts the target list from the RRSet and prepares
// the targets for the Endpoint objects, if needed.
func extractEndpointTargets(rrset *hcloud.ZoneRRSet) []string {
	targets := make([]string, len(rrset.Records))
	for idx, record := range rrset.Records {
		target := record.Value
		switch rrset.Type {
		case hcloud.ZoneRRSetTypeCNAME:
			target = fromHetznerHostname(rrset.Zone.Name, target)
		case hcloud.ZoneRRSetTypeMX:
			// MX records in Hetzner: "10 mail" (local) or "10 mail.beta.com." (external)
			// Convert to ExternalDNS format: "10 mail.zone.com" (FQDN without trailing dot)
			parts := strings.SplitN(target, " ", 2)
			if len(parts) == 2 {
				priority := parts[0]
				host := fromHetznerHostname(rrset.Zone.Name, parts[1])
				target = priority + " " + host
			} else {
				log.WithFields(log.Fields{
					"zone":   rrset.Zone.Name,
					"target": target,
				}).Warn("MX record from Hetzner API has unexpected format (expected 'priority hostname')")
			}
		}
		targets[idx] = target
	}
	return targets
}

// createEndpointFromRecord creates an endpoint from a record.
func createEndpointFromRecord(slash string, rrset *hcloud.ZoneRRSet) *endpoint.Endpoint {
	name := fmt.Sprintf("%s.%s", rrset.Name, rrset.Zone.Name)

	// root name is identified by @ and should be
	// translated to zone name for the endpoint entry.
	if rrset.Name == "@" {
		name = rrset.Zone.Name
	}

	// Handle local CNAMEs
	targets := extractEndpointTargets(rrset)
	ep := endpoint.NewEndpoint(name, string(rrset.Type), targets...)
	ep.ProviderSpecific = getProviderSpecific(slash, rrset.Labels)
	if rrset.TTL != nil {
		ep.RecordTTL = endpoint.TTL(*rrset.TTL)
	} else {
		ep.RecordTTL = endpoint.TTL(rrset.Zone.TTL)
	}
	log.WithFields(getEndpointLogFields(ep)).Debugf("Reading extracted endpoint %s", ep.DNSName)
	return ep
}

// endpointsByZoneID arranges the endpoints in a map by zone ID.
func endpointsByZoneID(zoneIDNameMapper zoneIDName, endpoints []*endpoint.Endpoint) map[int64][]*endpoint.Endpoint {
	endpointsByZoneID := make(map[int64][]*endpoint.Endpoint)

	for idx, ep := range endpoints {
		zoneID, _ := zoneIDNameMapper.FindZone(ep.DNSName)
		if zoneID == -1 {
			log.Warnf("Skipping record %s of type %s because no hosted zone matching record DNS Name was detected", ep.DNSName, ep.RecordType)
			continue
		} else {
			log.WithFields(getEndpointLogFields(ep)).Debugf("Reading endpoint %d for dividing by zone", idx)
		}
		endpointsByZoneID[zoneID] = append(endpointsByZoneID[zoneID], ep)
	}

	return endpointsByZoneID
}

// getMatchingRRSet returns the RRSet that matches an endpoint.
func getMatchingDomainRRSet(rrsets []*hcloud.ZoneRRSet, zoneName string, ep *endpoint.Endpoint) (*hcloud.ZoneRRSet, bool) {
	var name string
	if ep.DNSName != zoneName {
		name = strings.TrimSuffix(ep.DNSName, "."+zoneName)
	} else {
		name = "@"
	}

	for _, rrset := range rrsets {
		if rrset.Name == name && string(rrset.Type) == ep.RecordType {
			return rrset, true
		}
	}
	return nil, false
}

// getEndpointTTL returns a pointer to a value representing the endpoint TTL or
// nil if it is not configured.
func getEndpointTTL(ep *endpoint.Endpoint) *int {
	if !ep.RecordTTL.IsConfigured() {
		return nil
	}
	ttl := int(ep.RecordTTL)
	return &ttl
}

// getEndpointLogFields returns a loggable field map.
func getEndpointLogFields(ep *endpoint.Endpoint) log.Fields {
	return log.Fields{
		"DNSName":    ep.DNSName,
		"RecordType": ep.RecordType,
		"Targets":    ep.Targets.String(),
		"TTL":        int(ep.RecordTTL),
	}
}

// adjustEndpointTargets adjusts a serie of targets according to the
// specifications.
func adjustEndpointTargets(targets endpoint.Targets) (endpoint.Targets, error) {
	adjustedTargets := endpoint.Targets{}
	for _, target := range targets {
		adjustedTarget, err := makeEndpointTarget(target)
		if err != nil {
			return endpoint.Targets{}, err
		}
		adjustedTargets = append(adjustedTargets, adjustedTarget)
	}
	return adjustedTargets, nil
}

// getHetznerLabels returns the Hetzner-specific labels from the endpoint. The
// return map is always instantiated if there is no error.
func getHetznerLabels(slash string, ep *endpoint.Endpoint) (map[string]string, error) {
	ps := ep.ProviderSpecific
	if len(ps) == 0 {
		return nil, nil
	}
	return extractHetznerLabels(slash, ps)
}
