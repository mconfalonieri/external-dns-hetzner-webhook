/*
 * Changes Internals - Internal structures for processing changes.
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
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
)

func getRRSetRecordsString(records []hcloud.ZoneRRSetRecord) string {
	stringRecords := make([]string, len(records))
	for idx, record := range records {
		stringRecords[idx] = record.Value
	}
	return strings.Join(stringRecords, ";")
}

func getLabelMapString(labels map[string]string) string {
	arr := make([]string, 0)
	for k, v := range labels {
		arr = append(arr, k+"="+v)
	}
	return strings.Join(arr, ";")
}

// hetznerChangeCreate stores the information for a create request.
type hetznerChangeCreate struct {
	rrset *hcloud.ZoneRRSet
	opts  hcloud.ZoneRRSetAddRecordsOpts
}

// GetLogFields returns the log fields for this object.
func (cc hetznerChangeCreate) GetLogFields() log.Fields {
	return log.Fields{
		"zone":       cc.rrset.Zone.Name,
		"dnsName":    cc.rrset.Name,
		"recordType": string(cc.rrset.Type),
		"targets":    getRRSetRecordsString(cc.opts.Records),
		"ttl":        *cc.opts.TTL,
	}
}

// hetznerChangeUpdate stores the information for an update request.
type hetznerChangeUpdate struct {
	rrset       *hcloud.ZoneRRSet
	ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
	recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
	labels      map[string]string
}

// GetLogFields returns the log fields for this object. An asterisk indicate
// that the new value is shown.
func (cu hetznerChangeUpdate) GetLogFields() log.Fields {
	fields := log.Fields{
		"zone":       cu.rrset.Zone.Name,
		"dnsName":    cu.rrset.Name,
		"recordType": string(cu.rrset.Type),
	}
	if cu.ttlOpts != nil {
		fields["*ttl"] = cu.ttlOpts.TTL
	}
	if cu.recordsOpts != nil {
		fields["*targets"] = getRRSetRecordsString(cu.recordsOpts.Records)
	}
	if cu.labels != nil {
		fields["*labels"] = cu.labels
	}
	return fields
}

// hetznerChangeDelete stores the information for a delete request.
type hetznerChangeDelete struct {
	rrset *hcloud.ZoneRRSet
}

// GetLogFields returns the log fields for this object.
func (cd hetznerChangeDelete) GetLogFields() log.Fields {
	return log.Fields{
		"zone":       cd.rrset.Zone.Name,
		"dnsName":    cd.rrset.Name,
		"recordType": string(cd.rrset.Type),
	}
}
