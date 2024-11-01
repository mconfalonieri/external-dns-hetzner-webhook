/*
 * Changes - Processors - this file contains the code for processing the
 * changes.
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
package hetzner

import (
	"strings"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// adjustCNAMETarget fixes local CNAMEs.
func adjustCNAMETarget(domain string, target string) string {
	adjustedTarget := target
	if strings.HasSuffix(target, domain) {
		adjustedTarget = strings.TrimSuffix(target, domain)
	} else if strings.HasSuffix(target, domain+".") {
		adjustedTarget = strings.TrimSuffix(target, domain+".")
	} else if !strings.HasSuffix(target, ".") {
		adjustedTarget += "."
	}
	return adjustedTarget
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
			if ep.RecordType == "CNAME" {
				target = adjustCNAMETarget(zoneName, target)
			}
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
			changes.AddChangeCreate(zoneName, opts)
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

func processUpdateEndpoint(zoneID, zoneName string, matchingRecordsByTarget map[string]hdns.Record, ep *endpoint.Endpoint, changes *hetznerChanges) {
	// Generate create and delete actions based on existence of a record for each target.
	for _, target := range ep.Targets {
		if ep.RecordType == "CNAME" {
			target = adjustCNAMETarget(zoneName, target)
		}
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
			changes.AddChangeUpdate(zoneName, record, opts)

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
			changes.AddChangeCreate(zoneName, opts)
		}
	}
}

// cleanupRemainingTargets deletes the entries for the updates that are queued for creation.
func cleanupRemainingTargets(zoneID string, matchingRecordsByTarget map[string]hdns.Record, changes *hetznerChanges) {
	for _, record := range matchingRecordsByTarget {
		changes.AddChangeDelete(zoneID, record)
	}
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

		matchingRecordsByTarget := map[string]hdns.Record{}
		for _, r := range matchingRecords {
			matchingRecordsByTarget[r.Value] = r
		}

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

// processDeleteActionsByEndpoint processes delete actions for an endpoint.
func processDeleteActionsByEndpoint(zoneID string, matchingRecords []hdns.Record, ep *endpoint.Endpoint, changes *hetznerChanges) {
	for _, record := range matchingRecords {
		doDelete := false
		for _, t := range ep.Targets {
			v1 := t
			v2 := record.Value
			if ep.RecordType == endpoint.RecordTypeCNAME {
				v1 = strings.TrimSuffix(t, ".")
				v2 = strings.TrimSuffix(t, ".")
			}
			if v1 == v2 {
				doDelete = true
			}
		}

		if doDelete {
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
