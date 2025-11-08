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
	"strconv"
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

// hetznerChangeCreate stores the information for a create request.
type hetznerChangeCreate struct {
	zone *hcloud.Zone
	opts hcloud.ZoneRRSetCreateOpts
}

// GetLogFields returns the log fields for this object.
func (cc hetznerChangeCreate) GetLogFields() log.Fields {
	ttl := "unconfigured"
	if cc.opts.TTL != nil {
		ttl = strconv.FormatInt(int64(*cc.opts.TTL), 10)
	}
	return log.Fields{
		"zone":       cc.zone.Name,
		"dnsName":    cc.opts.Name,
		"recordType": string(cc.opts.Type),
		"targets":    getRRSetRecordsString(cc.opts.Records),
		"ttl":        ttl,
	}
}

// hetznerChangeUpdate stores the information for an update request.
type hetznerChangeUpdate struct {
	rrset       *hcloud.ZoneRRSet
	ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
	recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
	updateOpts  *hcloud.ZoneRRSetUpdateOpts
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
		ttl := "unconfigured"
		if cu.ttlOpts.TTL != nil {
			ttl = strconv.FormatInt(int64(*cu.ttlOpts.TTL), 10)
		}
		fields["*ttl"] = ttl
	}
	if cu.recordsOpts != nil {
		fields["*targets"] = getRRSetRecordsString(cu.recordsOpts.Records)
	}
	if cu.updateOpts != nil {
		fields["*labels"] = formatLabels(cu.updateOpts.Labels)
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
