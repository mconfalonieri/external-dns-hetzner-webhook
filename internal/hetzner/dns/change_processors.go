/*
 * Change processors - this file contains the code for processing changes and
 * queue them.
 *
 * Copyright 2023 Marco Confalonieri.
 * Copyright 2017 The Kubernetes Authors.
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
	"strconv"
	"strings"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// adjustCNAMETarget fixes local CNAME targets. It ensures that targets
// matching the domain are stripped of the domain parts and that "external"
// targets end with a dot.
//
// Hetzner DNS convention: local hostnames have NO trailing dot, external DO.
// See: https://docs.hetzner.com/dns-console/dns/record-types/mx-record/
func adjustCNAMETarget(domain string, target string) string {
	adjustedTarget := target
	if strings.HasSuffix(target, "."+domain) {
		adjustedTarget = strings.TrimSuffix(target, "."+domain)
	} else if strings.HasSuffix(target, "."+domain+".") {
		adjustedTarget = strings.TrimSuffix(target, "."+domain+".")
	} else if !strings.HasSuffix(target, ".") {
		adjustedTarget += "."
	}
	return adjustedTarget
}

// adjustMXTarget adjusts MX record target to Hetzner DNS format.
// MX target format from ExternalDNS: "10 mail.example.com"
// Hetzner expects: "10 mail" (local) or "10 mail.other.com." (external with dot)
func adjustMXTarget(domain string, target string) string {
	parts := strings.SplitN(target, " ", 2)
	if len(parts) != 2 {
		log.WithFields(log.Fields{
			"target": target,
		}).Warn("MX target has invalid format (expected 'priority hostname')")
		return target
	}
	priority := parts[0]
	host := parts[1]

	// Validate priority is numeric
	if _, err := strconv.Atoi(priority); err != nil {
		log.WithFields(log.Fields{
			"target":   target,
			"priority": priority,
		}).Warn("MX priority is not a valid integer")
		return target
	}

	// Handle apex record (host equals domain)
	hostNoDot := strings.TrimSuffix(host, ".")
	if hostNoDot == domain {
		return priority + " @"
	}

	// Use existing CNAME logic for hostname
	return priority + " " + adjustCNAMETarget(domain, host)
}

// adjustTarget adjusts the target depending on its type
func adjustTarget(domain, recordType, target string) string {
	switch recordType {
	case "CNAME":
		target = adjustCNAMETarget(domain, target)
	case "MX":
		target = adjustMXTarget(domain, target)
	}
	return target
}

// processCreateActionsByZone processes the create actions for one zone.
func processCreateActionsByZone(zoneID, zoneName string, records []hdns.Record, endpoints []*endpoint.Endpoint, changes *hetznerChanges) {
	for _, ep := range endpoints {
		// Warn if there are existing records since we expect to create only new records.
		matchingRecords := getMatchingDomainRecords(records, zoneName, ep)
		if len(matchingRecords) > 0 {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Preexisting records exist which should not exist for creation actions.")
		}

		for _, target := range ep.Targets {
			target = adjustTarget(zoneName, ep.RecordType, target)
			opts := &hdns.RecordCreateOpts{
				Name:  makeEndpointName(zoneName, ep.DNSName),
				Ttl:   getEndpointTTL(ep),
				Type:  hdns.RecordType(ep.RecordType),
				Value: target,
				Zone: &hdns.Zone{
					ID:   zoneID,
					Name: zoneName,
				},
			}
			changes.AddChangeCreate(zoneID, opts)
		}
	}
}

// processCreateActions processes the create requests.
func processCreateActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	createsByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
) {
	// Process endpoints that need to be created.
	for zoneID, endpoints := range createsByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping domain, no creates found.")
			continue
		}
		records := recordsByZoneID[zoneID]
		processCreateActionsByZone(zoneID, zoneName, records, endpoints, changes)
	}
}

// processUpdateEndpoint processes the update requests for an endpoint.
func processUpdateEndpoint(zoneID, zoneName string, matchingRecordsByTarget map[string]hdns.Record, ep *endpoint.Endpoint, changes *hetznerChanges) {
	// Generate create and delete actions based on existence of a record for each target.
	for _, target := range ep.Targets {
		target = adjustTarget(zoneName, ep.RecordType, target)
		if record, ok := matchingRecordsByTarget[target]; ok {
			opts := &hdns.RecordUpdateOpts{
				Name:  makeEndpointName(zoneName, ep.DNSName),
				Ttl:   getEndpointTTL(ep),
				Type:  hdns.RecordType(ep.RecordType),
				Value: target,
				Zone: &hdns.Zone{
					ID:   zoneID,
					Name: zoneName,
				},
			}
			changes.AddChangeUpdate(zoneID, record, opts)

			// Updates are removed from this map.
			delete(matchingRecordsByTarget, target)
		} else {
			// Record did not previously exist, create new 'target'
			opts := &hdns.RecordCreateOpts{
				Name:  makeEndpointName(zoneName, ep.DNSName),
				Ttl:   getEndpointTTL(ep),
				Type:  hdns.RecordType(ep.RecordType),
				Value: target,
				Zone: &hdns.Zone{
					ID:   zoneID,
					Name: zoneName,
				},
			}
			changes.AddChangeCreate(zoneID, opts)
		}
	}
}

// cleanupRemainingTargets deletes the entries for the updates that are queued for creation.
func cleanupRemainingTargets(zoneID string, matchingRecordsByTarget map[string]hdns.Record, changes *hetznerChanges) {
	for _, record := range matchingRecordsByTarget {
		changes.AddChangeDelete(zoneID, record)
	}
}

// getMatchingRecordsByTarget organizes a slice of targets in a map with the
// target as key.
func getMatchingRecordsByTarget(records []hdns.Record) map[string]hdns.Record {
	recordsMap := make(map[string]hdns.Record, 0)
	for _, r := range records {
		recordsMap[r.Value] = r
	}
	return recordsMap
}

// processUpdateActionsByZone processes update actions for a single zone.
func processUpdateActionsByZone(zoneID, zoneName string, records []hdns.Record, endpoints []*endpoint.Endpoint, changes *hetznerChanges) {
	for _, ep := range endpoints {
		matchingRecords := getMatchingDomainRecords(records, zoneName, ep)

		if len(matchingRecords) == 0 {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Planning an update but no existing records found.")
		}

		matchingRecordsByTarget := getMatchingRecordsByTarget(matchingRecords)

		processUpdateEndpoint(zoneID, zoneName, matchingRecordsByTarget, ep, changes)

		// Any remaining records have been removed, delete them
		cleanupRemainingTargets(zoneID, matchingRecordsByTarget, changes)
	}
}

// processUpdateActions processes the update requests.
func processUpdateActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	updatesByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
) {
	// Generate creates and updates based on existing
	for zoneID, endpoints := range updatesByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no updates found.")
			continue
		}

		records := recordsByZoneID[zoneID]
		processUpdateActionsByZone(zoneID, zoneName, records, endpoints, changes)

	}
}

// targetsMatch determines if a record matches one of the endpoint's targets.
func targetsMatch(record hdns.Record, ep *endpoint.Endpoint) bool {
	for _, t := range ep.Targets {
		endpointTarget := t
		recordTarget := record.Value
		domain := record.Zone.Name
		endpointTarget = adjustTarget(domain, ep.RecordType, t)
		if endpointTarget == recordTarget {
			return true
		}
	}
	return false
}

// processDeleteActionsByEndpoint processes delete actions for an endpoint.
func processDeleteActionsByEndpoint(zoneID string, matchingRecords []hdns.Record, ep *endpoint.Endpoint, changes *hetznerChanges) {
	for _, record := range matchingRecords {
		if targetsMatch(record, ep) {
			changes.AddChangeDelete(zoneID, record)
		}
	}
}

// processDeleteActionsByZone processes delete actions for a single zone.
func processDeleteActionsByZone(zoneID, zoneName string, records []hdns.Record, endpoints []*endpoint.Endpoint, changes *hetznerChanges) {
	for _, ep := range endpoints {
		matchingRecords := getMatchingDomainRecords(records, zoneName, ep)

		if len(matchingRecords) == 0 {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Records to delete not found.")
		}
		processDeleteActionsByEndpoint(zoneID, matchingRecords, ep, changes)
	}
}

// processDeleteActions processes the delete requests.
func processDeleteActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	deletesByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
) {
	for zoneID, endpoints := range deletesByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no deletes found.")
			continue
		}

		records := recordsByZoneID[zoneID]
		processDeleteActionsByZone(zoneID, zoneName, records, endpoints, changes)

	}
}
