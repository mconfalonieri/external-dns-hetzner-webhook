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
package hetznercloud

import (
	"strings"

	"sigs.k8s.io/external-dns/endpoint"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// adjustCNAMETarget fixes local CNAME targets. It ensures that targets
// matching the domain are stripped of the domain parts and that "external"
// targets end with a dot.
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

// processCreateActionsByZone processes the create actions for one zone.
func processCreateActionsByZone(zoneID int64, zoneName string, rrsets []*hcloud.ZoneRRSet, endpoints []*endpoint.Endpoint, changes *hetznerChanges) {
	for _, ep := range endpoints {
		// Warn if there are existing records since we expect to create only new records.
		if matchingRRSet, _ := getMatchingDomainRRSet(rrsets, zoneName, ep); matchingRRSet != nil {
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
			opts := &hcloud.RecordCreateOpts{
				Name:  makeEndpointName(zoneName, ep.DNSName),
				Ttl:   getEndpointTTL(ep),
				Type:  hdns.RecordType(ep.RecordType),
				Value: target,
				Zone: &hcloud.Zone{
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
	zoneIDNameMapper zoneIDName,
	rrSetsByZoneID map[int64][]*hcloud.ZoneRRSet,
	createsByZoneID map[int64][]*endpoint.Endpoint,
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
		rrsets := rrSetsByZoneID[zoneID]
		processCreateActionsByZone(zoneID, zoneName, rrsets, endpoints, changes)
	}
}

func processUpdateEndpoint(mRRSet *hcloud.ZoneRRSet, ep *endpoint.Endpoint, changes *hetznerChanges) {
	// Generate
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
func processUpdateActionsByZone(zoneName string, rrsets []*hcloud.ZoneRRSet, endpoints []*endpoint.Endpoint, changes *hetznerChanges) {
	for _, ep := range endpoints {
		if mRRSet, _ := getMatchingDomainRRSet(rrsets, zoneName, ep); mRRSet != nil {
			processUpdateEndpoint(mRRSet, ep, changes)
		} else {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Planning an update but no existing records found.")
		}

	}
}

// processUpdateActions processes the update requests.
func processUpdateActions(
	zoneIDNameMapper zoneIDName,
	recordsByZoneID map[int64][]*hcloud.ZoneRRSet,
	updatesByZoneID map[int64][]*endpoint.Endpoint,
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
		if ep.RecordType == endpoint.RecordTypeCNAME {
			domain := record.Zone.Name
			endpointTarget = adjustCNAMETarget(domain, t)
		}
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
	zoneIDNameMapper zoneIDName,
	recordsByZoneID map[int64][]*hcloud.ZoneRRSet,
	deletesByZoneID map[int64][]*endpoint.Endpoint,
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
