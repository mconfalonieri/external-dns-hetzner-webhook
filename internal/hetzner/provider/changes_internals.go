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
package provider

import (
	"external-dns-hetzner-webhook/internal/hetzner/model"

	log "github.com/sirupsen/logrus"
)

// hetznerChangeCreate stores the information for a create request.
type hetznerChangeCreate model.Record

// GetLogFields returns the log fields for this object.
func (cc hetznerChangeCreate) GetLogFields() log.Fields {
	return log.Fields{
		"domain":     cc.Zone.Name,
		"zoneID":     cc.Zone.ID,
		"dnsName":    cc.Name,
		"recordType": cc.Type,
		"value":      cc.Value,
		"ttl":        cc.TTL,
	}
}

// hetznerChangeUpdate stores the information for an update request.
type hetznerChangeUpdate model.Record

// GetLogFields returns the log fields for this object. An asterisk indicate
// that the new value is shown.
func (cu hetznerChangeUpdate) GetLogFields() log.Fields {
	return log.Fields{
		"domain":      cu.Zone.Name,
		"zoneID":      cu.Zone.ID,
		"recordID":    cu.ID,
		"*dnsName":    cu.Name,
		"*recordType": cu.Type,
		"*value":      cu.Value,
		"*ttl":        cu.TTL,
	}
}

// hetznerChangeDelete stores the information for a delete request.
type hetznerChangeDelete model.Record

// GetLogFields returns the log fields for this object.
func (cd hetznerChangeDelete) GetLogFields() log.Fields {
	return log.Fields{
		"domain":     cd.Zone.Name,
		"zoneID":     cd.Zone.ID,
		"dnsName":    cd.Name,
		"recordType": cd.Type,
		"value":      cd.Value,
	}
}
