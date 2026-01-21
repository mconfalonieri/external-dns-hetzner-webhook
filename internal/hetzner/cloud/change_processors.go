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

// extractRRSetRecords extracts the records from an endpoint.
func extractRRSetRecords(zoneName string, ep *endpoint.Endpoint) []hcloud.ZoneRRSetRecord {
	targets := []string(ep.Targets)
	records := make([]hcloud.ZoneRRSetRecord, len(targets))
	for idx, target := range targets {
		if ep.RecordType == "CNAME" {
			target = adjustCNAMETarget(zoneName, target)
		}
		records[idx] = hcloud.ZoneRRSetRecord{
			Value: target,
		}
	}
	return records
}

// processCreateActionsByZone processes the create actions for one zone.
func processCreateActionsByZone(zone *hcloud.Zone, rrsets []*hcloud.ZoneRRSet, endpoints []*endpoint.Endpoint, changes changesRunner) {
	zoneName := zone.Name
	for _, ep := range endpoints {
		// If there is an existing record we refuse to act.
		if matchingRRSet, _ := getMatchingDomainRRSet(rrsets, zoneName, ep); matchingRRSet != nil {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Planning a creation but an existing record was found.")
		} else {
			var err error
			var labels map[string]string
			slash, labelsSupported := changes.GetSlash()
			if labels, err = getHetznerLabels(slash, ep); err != nil {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
				}).Warnf("Labels will be ignored due to a parsing error: %s", err.Error())
			} else if !labelsSupported && len(labels) > 0 {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
				}).Warn("Labels are ignored in BULK_MODE.")
			}
			opts := hcloud.ZoneRRSetCreateOpts{
				Name:    makeEndpointName(zoneName, ep.DNSName),
				Type:    hcloud.ZoneRRSetType(ep.RecordType),
				TTL:     getEndpointTTL(ep),
				Records: extractRRSetRecords(zoneName, ep),
				Labels:  labels,
			}
			changes.AddChangeCreate(zone, opts)
		}
	}
}

// processCreateActions processes the create requests.
func processCreateActions(
	zoneIDNameMapper zoneIDName,
	rrSetsByZoneID map[int64][]*hcloud.ZoneRRSet,
	createsByZoneID map[int64][]*endpoint.Endpoint,
	changes changesRunner,
) {
	// Process endpoints that need to be created.
	for zoneID, endpoints := range createsByZoneID {
		zone := zoneIDNameMapper[zoneID]
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zone.Name,
			}).Debug("Skipping domain, no creates found.")
			continue
		}
		rrsets := rrSetsByZoneID[zoneID]
		processCreateActionsByZone(zone, rrsets, endpoints, changes)
	}
}

// sameZoneRRSetRecords returns true if two arrays contains the same elements
// and false otherwise. Please note that this implementation purposely excludes
// the comments from the comparison.
func sameZoneRRSetRecords(first, second []hcloud.ZoneRRSetRecord) bool {
	// If the length is different, it is false.
	if len(first) != len(second) {
		return false
	}
	// Build a map "reversing" index and record. For the latter, we are only
	// interested in the Value field.
	second_map := make(map[string]int, len(second))
	for i, r := range second {
		second_map[r.Value] = i
	}

	// Delete from second_map the values found in first
	for _, r := range first {
		value := r.Value
		delete(second_map, value)
	}

	// If all elements in second_map are deleted, first and second have the
	// same elements, as we already ruled out different lengths.
	return len(second_map) == 0
}

// ensureStringMap ensures that a map is instantiated.
func ensureStringMap(m map[string]string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// equalStringMaps checks two maps for equality. We don't want to rely on
// reflection due to performance costs.
func equalStringMaps(first, second map[string]string) bool {
	first = ensureStringMap(first)
	second = ensureStringMap(second)
	// Check length
	if len(first) != len(second) {
		return false
	}
	// Check contents
	for k, v1 := range first {
		if v2, ok := second[k]; ok {
			if v1 != v2 {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func processUpdateEndpoint(mRRSet *hcloud.ZoneRRSet, ep *endpoint.Endpoint, changes changesRunner) {
	zone := mRRSet.Zone
	zoneName := zone.Name

	// The arguments that we want to fill.
	var (
		ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts  = nil
		recordsOpts *hcloud.ZoneRRSetSetRecordsOpts = nil
		updateOpts  *hcloud.ZoneRRSetUpdateOpts     = nil
	)

	// Check if we need to update the TTL. We do not update an unconfigured TTL
	epTTL := getEndpointTTL(ep)
	rrSetTTL := mRRSet.TTL
	if epTTL != nil && (rrSetTTL == nil || *rrSetTTL != *epTTL) {
		newTTL := *epTTL
		ttlOpts = &hcloud.ZoneRRSetChangeTTLOpts{
			TTL: &newTTL,
		}
	}

	// Check if we need to update the records
	records := extractRRSetRecords(zoneName, ep)
	if !sameZoneRRSetRecords(records, mRRSet.Records) {
		recordsOpts = &hcloud.ZoneRRSetSetRecordsOpts{
			Records: records,
		}
	}

	slash, labelsSupported := changes.GetSlash()
	// Check if we need to update the labels
	labels, err := getHetznerLabels(slash, ep)

	if err != nil {
		log.WithFields(log.Fields{
			"zoneName":   zoneName,
			"dnsName":    ep.DNSName,
			"recordType": ep.RecordType,
		}).Warnf("Labels will be ignored for a parsing error: %s", err.Error())
	} else if !labelsSupported && len(labels) > 0 {
		log.WithFields(log.Fields{
			"zoneName":   zoneName,
			"dnsName":    ep.DNSName,
			"recordType": ep.RecordType,
		}).Warn("Labels are ignored in BULK_MODE.")
	} else if !equalStringMaps(labels, mRRSet.Labels) {
		log.Debugf("Updating labels to %s", formatLabels(labels))
		updateOpts = &hcloud.ZoneRRSetUpdateOpts{
			Labels: labels,
		}
	}

	changes.AddChangeUpdate(mRRSet, ttlOpts, recordsOpts, updateOpts)
}

// processUpdateActionsByZone processes update actions for a single zone.
func processUpdateActionsByZone(zone *hcloud.Zone, rrsets []*hcloud.ZoneRRSet, endpoints []*endpoint.Endpoint, changes changesRunner) {
	zoneName := zone.Name
	for _, ep := range endpoints {
		mRRSet, found := getMatchingDomainRRSet(rrsets, zoneName, ep)
		if !found {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("Planning an update but no existing records found.")
		} else {
			mRRSet.Zone = zone
			processUpdateEndpoint(mRRSet, ep, changes)
		}

	}
}

// processUpdateActions processes the update requests.
func processUpdateActions(
	zoneIDNameMapper zoneIDName,
	rrSetsByZoneID map[int64][]*hcloud.ZoneRRSet,
	updatesByZoneID map[int64][]*endpoint.Endpoint,
	changes changesRunner,
) {
	// Generate creates and updates based on existing
	for zoneID, endpoints := range updatesByZoneID {
		zone := zoneIDNameMapper[zoneID]
		zoneName := zone.Name
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no updates found.")
			continue
		}
		rrsets := rrSetsByZoneID[zoneID]
		processUpdateActionsByZone(zone, rrsets, endpoints, changes)

	}
}

// processDeleteActionsByZone processes delete actions for a single zone.
func processDeleteActionsByZone(zone *hcloud.Zone, rrsets []*hcloud.ZoneRRSet, endpoints []*endpoint.Endpoint, changes changesRunner) {
	zoneName := zone.Name
	for _, ep := range endpoints {
		mRRSet, found := getMatchingDomainRRSet(rrsets, zoneName, ep)
		if !found {
			log.WithFields(log.Fields{
				"zoneName":   zoneName,
				"dnsName":    ep.DNSName,
				"recordType": ep.RecordType,
			}).Warn("RRSet to delete not found.")
		} else {
			mRRSet.Zone = zone
			changes.AddChangeDelete(mRRSet)
		}

	}
}

// processDeleteActions processes the delete requests.
func processDeleteActions(
	zoneIDNameMapper zoneIDName,
	rrSetsByZoneID map[int64][]*hcloud.ZoneRRSet,
	deletesByZoneID map[int64][]*endpoint.Endpoint,
	changes changesRunner,
) {
	for zoneID, endpoints := range deletesByZoneID {
		zone := zoneIDNameMapper[zoneID]
		zoneName := zone.Name
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no deletes found.")
			continue
		}

		rrsets := rrSetsByZoneID[zoneID]
		processDeleteActionsByZone(zone, rrsets, endpoints, changes)

	}
}
