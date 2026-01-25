/*
 * Endpoints - functions for handling and transforming endpoints.
 *
 * Copyright 2024 Marco Confalonieri.
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
package hetznerdns

import (
	"fmt"
	"strings"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
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
// requirements:
//   - A-Records should respect ignored networks and should only contain IPv4
//     entries.
func makeEndpointTarget(domain, entryTarget string, _ string) string {
	if domain == "" {
		return entryTarget
	}

	// Trim the trailing dot
	adjustedTarget := strings.TrimSuffix(entryTarget, ".")

	return adjustedTarget
}

// mergeEndpointsByNameType merges Endpoints with the same Name and Type into a
// single endpoint with multiple Targets.
func mergeEndpointsByNameType(endpoints []*endpoint.Endpoint) []*endpoint.Endpoint {
	endpointsByNameType := map[string][]*endpoint.Endpoint{}

	for _, e := range endpoints {
		key := fmt.Sprintf("%s-%s", e.DNSName, e.RecordType)
		endpointsByNameType[key] = append(endpointsByNameType[key], e)
	}

	// If no merge occurred, just return the existing endpoints.
	if len(endpointsByNameType) == len(endpoints) {
		return endpoints
	}

	// Otherwise, construct a new list of endpoints with the endpoints merged.
	var result []*endpoint.Endpoint
	for _, endpoints := range endpointsByNameType {
		dnsName := endpoints[0].DNSName
		recordType := endpoints[0].RecordType

		targets := make([]string, len(endpoints))
		for i, e := range endpoints {
			targets[i] = e.Targets[0]
		}

		e := endpoint.NewEndpoint(dnsName, recordType, targets...)
		e.RecordTTL = endpoints[0].RecordTTL
		result = append(result, e)
	}

	return result
}

// createEndpointFromRecord creates an endpoint from a record.
func createEndpointFromRecord(r hdns.Record) *endpoint.Endpoint {
	name := fmt.Sprintf("%s.%s", r.Name, r.Zone.Name)

	// root name is identified by @ and should be
	// translated to zone name for the endpoint entry.
	if r.Name == "@" {
		name = r.Zone.Name
	}

	// Handle local CNAMEs
	target := r.Value
	zoneName := r.Zone.Name
	switch r.Type {
	case hdns.RecordTypeCNAME:
		target = fromHetznerHostname(zoneName, target)
	case hdns.RecordTypeMX:
		// MX records in Hetzner: "10 mail" (local) or "10 mail.beta.com." (external)
		// Convert to ExternalDNS format: "10 mail.zone.com" (FQDN without trailing dot)
		parts := strings.SplitN(target, " ", 2)
		if len(parts) == 2 {
			priority := parts[0]
			host := fromHetznerHostname(zoneName, parts[1])
			target = priority + " " + host
		} else {
			log.WithFields(log.Fields{
				"zone":   zoneName,
				"target": target,
			}).Warn("MX record from Hetzner API has unexpected format (expected 'priority hostname')")
		}
	}
	ep := endpoint.NewEndpoint(name, string(r.Type), target)
	ep.RecordTTL = endpoint.TTL(r.Ttl)
	return ep
}

// endpointsByZoneID arranges the endpoints in a map by zone ID.
func endpointsByZoneID(zoneIDNameMapper provider.ZoneIDName, endpoints []*endpoint.Endpoint) map[string][]*endpoint.Endpoint {
	endpointsByZoneID := make(map[string][]*endpoint.Endpoint)

	for idx, ep := range endpoints {
		zoneID, _ := zoneIDNameMapper.FindZone(ep.DNSName)
		if zoneID == "" {
			log.Debugf("Skipping record %d (%s) because no hosted zone matching record DNS Name was detected", idx, ep.DNSName)
			continue
		} else {
			log.WithFields(getEndpointLogFields(ep)).Debugf("Reading endpoint %d for dividing by zone", idx)
		}
		endpointsByZoneID[zoneID] = append(endpointsByZoneID[zoneID], ep)
	}

	return endpointsByZoneID
}

// getMatchingDomainRecords returns the records that match an endpoint.
func getMatchingDomainRecords(records []hdns.Record, zoneName string, ep *endpoint.Endpoint) []hdns.Record {
	var name string
	if len(ep.ProviderSpecific) > 0 {
		log.Warnf("Ignoring provider-specific directives in endpoint [%s] of type [%s].", ep.DNSName, ep.RecordType)
	}
	if ep.DNSName != zoneName {
		name = strings.TrimSuffix(ep.DNSName, "."+zoneName)
	} else {
		name = "@"
	}

	var result []hdns.Record
	for _, r := range records {
		if r.Name == name && string(r.Type) == ep.RecordType {
			result = append(result, r)
		}
	}
	return result
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
