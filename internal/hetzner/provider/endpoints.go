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
package provider

import (
	"external-dns-hetzner-webhook/internal/hetzner/model"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
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
func createEndpointFromRecord(r model.Record) *endpoint.Endpoint {
	name := fmt.Sprintf("%s.%s", r.Name, r.Zone.Name)

	// root name is identified by @ and should be
	// translated to zone name for the endpoint entry.
	if r.Name == "@" {
		name = r.Zone.Name
	}

	// Handle local CNAMEs
	target := r.Value
	if r.Type == "CNAME" && !strings.HasSuffix(r.Value, ".") {
		target = fmt.Sprintf("%s.%s.", r.Value, r.Zone.Name)
	}
	ep := endpoint.NewEndpoint(name, string(r.Type), target)
	ep.RecordTTL = endpoint.TTL(r.TTL)
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
func getMatchingDomainRecords(records []model.Record, zoneName string, ep *endpoint.Endpoint) []model.Record {
	var name string
	if ep.DNSName != zoneName {
		name = strings.TrimSuffix(ep.DNSName, "."+zoneName)
	} else {
		name = "@"
	}

	var result []model.Record
	for _, r := range records {
		if r.Name == name && r.Type == ep.RecordType {
			result = append(result, r)
		}
	}
	return result
}

// getEndpointTTL returns a pointer to a value representing the endpoint TTL or
// nil if it is not configured.
func getEndpointTTL(ep *endpoint.Endpoint) int {
	if !ep.RecordTTL.IsConfigured() {
		return -1
	}
	ttl := int(ep.RecordTTL)
	return ttl
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
