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
package hetzner

import (
	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// hetznerChangeCreate stores the information for a create request.
type hetznerChangeCreate struct {
	ZoneID  string
	Options *hdns.RecordCreateOpts
}

// GetLogFields returns the log fields for this object.
func (cc hetznerChangeCreate) GetLogFields() log.Fields {
	return log.Fields{
		"domain":     cc.Options.Zone.Name,
		"zoneID":     cc.ZoneID,
		"dnsName":    cc.Options.Name,
		"recordType": string(cc.Options.Type),
		"value":      cc.Options.Value,
		"ttl":        *cc.Options.Ttl,
	}
}

// hetznerChangeUpdate stores the information for an update request.
type hetznerChangeUpdate struct {
	ZoneID  string
	Record  hdns.Record
	Options *hdns.RecordUpdateOpts
}

// GetLogFields returns the log fields for this object. An asterisk indicate
// that the new value is shown.
func (cu hetznerChangeUpdate) GetLogFields() log.Fields {
	return log.Fields{
		"domain":      cu.Record.Zone.Name,
		"zoneID":      cu.ZoneID,
		"recordID":    cu.Record.ID,
		"*dnsName":    cu.Options.Name,
		"*recordType": string(cu.Options.Type),
		"*value":      cu.Options.Value,
		"*ttl":        *cu.Options.Ttl,
	}
}

// hetznerChangeDelete stores the information for a delete request.
type hetznerChangeDelete struct {
	ZoneID string
	Record hdns.Record
}

// GetLogFields returns the log fields for this object.
func (cd hetznerChangeDelete) GetLogFields() log.Fields {
	return log.Fields{
		"domain":     cd.Record.Zone.Name,
		"zoneID":     cd.ZoneID,
		"dnsName":    cd.Record.Name,
		"recordType": string(cd.Record.Type),
		"value":      cd.Record.Value,
	}
}
