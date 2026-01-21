/*
 * Changes - Code for storing changes and sending them to the DNS API.
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
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
)

type zoneChanges struct {
	creates []*hetznerChangeCreate
	updates []*hetznerChangeUpdate
	deletes []*hetznerChangeDelete
}

// bulkChanges contains all changes to apply to DNS using the bulk system. This
// method exports the BIND zone file, applies the changes and re-uploads it,
// therefore using always exactly two calls per zone.
type bulkChanges struct {
	dnsClient  apiClient
	dryRun     bool
	defaultTTL int
	slash      string

	zones   map[int64]*hcloud.Zone
	changes map[int64]*zoneChanges
}

// NewBulkChanges creates a new bulkChanges object.
func NewBulkChanges(dnsClient apiClient, dryRun bool, defaultTTL int, slash string) *bulkChanges {
	return &bulkChanges{
		dnsClient:  dnsClient,
		dryRun:     dryRun,
		defaultTTL: defaultTTL,
		slash:      slash,
		changes:    make(map[int64]*zoneChanges, 0),
	}
}

// empty returns true if there are no changes left.
func (c bulkChanges) empty() bool {
	return len(c.changes) == 0
}

// GetSlash returns the escape sequence for slash and true (labels supported)
// as the second parameter.
func (c bulkChanges) GetSlash() (string, bool) {
	return c.slash, true
}

// getZoneChanges returns or creates the appropriate zoneChanges object for the
// zone.
func (c *bulkChanges) getZoneChanges(zone *hcloud.Zone) *zoneChanges {
	var zc *zoneChanges
	if _, ok := c.zones[zone.ID]; !ok {
		zc := &zoneChanges{}
		c.zones[zone.ID] = zone
		c.changes[zone.ID] = zc
	} else {
		zc = c.changes[zone.ID]
	}
	return zc
}

// AddChangeCreate adds a new creation entry to the current object.
func (c *bulkChanges) AddChangeCreate(zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts) {
	changeCreate := &hetznerChangeCreate{
		zone: zone,
		opts: opts,
	}
	zc := c.getZoneChanges(zone)
	zc.creates = append(zc.creates, changeCreate)
}

// AddChangeUpdate adds a new update entry to the current object.
func (c *bulkChanges) AddChangeUpdate(rrset *hcloud.ZoneRRSet, ttlOpts *hcloud.ZoneRRSetChangeTTLOpts, recordsOpts *hcloud.ZoneRRSetSetRecordsOpts, updateOpts *hcloud.ZoneRRSetUpdateOpts) {
	changeUpdate := &hetznerChangeUpdate{
		rrset:       rrset,
		ttlOpts:     ttlOpts,
		recordsOpts: recordsOpts,
		updateOpts:  updateOpts,
	}
	zc := c.getZoneChanges(rrset.Zone)
	zc.updates = append(zc.updates, changeUpdate)
}

// AddChangeDelete adds a new delete entry to the current object.
func (c *bulkChanges) AddChangeDelete(rrset *hcloud.ZoneRRSet) {
	changeDelete := &hetznerChangeDelete{
		rrset: rrset,
	}
	zc := c.getZoneChanges(rrset.Zone)
	zc.deletes = append(zc.deletes, changeDelete)
}

// applyChangesZone applies changes to a zone.
func (c bulkChanges) applyChangesZone(ctx context.Context, zone *hcloud.Zone) {
	/*
		zfr, _, err := c.dnsClient.ExportZonefile(ctx, zone)
		if err != nil {
			log.WithFields(log.Fields{
				"zoneName": zone.Name,
			}).Errorf("Error while exporting zonefile: %w", err)
			return
		}

		zp := dns.NewZoneParser(strings.NewReader(zfr.Zonefile), origin, file)
		if err != nil {
			log.WithFields(log.Fields{
				"zoneName": zone.Name,
			}).Errorf("Error while reading the exported zonefile: %w", err)
			return
		}*/

}

// ApplyChanges applies the planned changes.
func (c bulkChanges) ApplyChanges(ctx context.Context) error {
	// No changes = nothing to do.
	if c.empty() {
		log.Debug("No changes to be applied found.")
		return nil
	}
	for _, z := range c.zones {
		c.applyChangesZone(ctx, z)
	}
	return nil
}
