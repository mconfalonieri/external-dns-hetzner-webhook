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
package hetznercloud

import (
	"fmt"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/idna"
	"sigs.k8s.io/external-dns/endpoint"
)

// makeEndpointName makes a endpoint name that conforms to Hetzner DNS
// requirements:
//   - the adjusted name must be without domain,
//   - records at root of the zone have `@` as the name.
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
	for _, record := range rrset.Records {
		target := record.Value
		if rrset.Type == hcloud.ZoneRRSetTypeCNAME && !strings.HasSuffix(target, ".") {
			target = fmt.Sprintf("%s.%s.", target, rrset.Zone.Name)
		}
	}
	return targets
}

// createEndpointFromRecord creates an endpoint from a record.
func createEndpointFromRecord(rrset *hcloud.ZoneRRSet) *endpoint.Endpoint {
	name := fmt.Sprintf("%s.%s", rrset.Name, rrset.Zone.Name)

	// root name is identified by @ and should be
	// translated to zone name for the endpoint entry.
	if rrset.Name == "@" {
		name = rrset.Zone.Name
	}

	// Handle local CNAMEs
	targets := extractEndpointTargets(rrset)
	ep := endpoint.NewEndpoint(name, string(rrset.Type), targets...)
	ep.RecordTTL = endpoint.TTL(*rrset.TTL)
	return ep
}

// endpointsByZoneID arranges the endpoints in a map by zone ID.
func endpointsByZoneID(zoneIDNameMapper zoneIDName, endpoints []*endpoint.Endpoint) map[int64][]*endpoint.Endpoint {
	endpointsByZoneID := make(map[int64][]*endpoint.Endpoint)

	for idx, ep := range endpoints {
		zoneID, _ := zoneIDNameMapper.FindZone(ep.DNSName)
		if zoneID == -1 {
			log.Debugf("Skipping record %d (%s) because no hosted zone matching record DNS Name was detected", idx, ep.DNSName)
			continue
		} else {
			log.WithFields(getEndpointLogFields(ep)).Debugf("Reading endpoint %d for dividing by zone", idx)
		}
		endpointsByZoneID[zoneID] = append(endpointsByZoneID[zoneID], ep)
	}

	return endpointsByZoneID
}

// getMatchingRRSet returns the RRSet that matches an endpoint.
func getMatchingDomainRRSet(rrsets []*hcloud.ZoneRRSet, zoneName string, ep *endpoint.Endpoint) (*hcloud.ZoneRRSet, error) {
	var name string
	if ep.DNSName != zoneName {
		name = strings.TrimSuffix(ep.DNSName, "."+zoneName)
	} else {
		name = "@"
	}

	for _, rrset := range rrsets {
		if rrset.Name == name && string(rrset.Type) == ep.RecordType {
			return rrset, nil
		}
	}
	return nil, fmt.Errorf("cannot find an RRSet matching name=%s and type=%s", ep.DNSName, ep.RecordType)
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
