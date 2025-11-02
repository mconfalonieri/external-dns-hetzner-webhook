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
	"time"

	"external-dns-hetzner-webhook/internal/metrics"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
)

// hetznerChange contains all changes to apply to DNS.
type hetznerChanges struct {
	dryRun     bool
	defaultTTL int

	creates []*hetznerChangeCreate
	updates []*hetznerChangeUpdate
	deletes []*hetznerChangeDelete
}

// empty returns true if there are no changes left.
func (c *hetznerChanges) empty() bool {
	return len(c.creates) == 0 && len(c.updates) == 0 && len(c.deletes) == 0
}

// AddChangeCreate adds a new creation entry to the current object.
func (c *hetznerChanges) AddChangeCreate(zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts) {
	changeCreate := &hetznerChangeCreate{
		zone: zone,
		opts: opts,
	}
	c.creates = append(c.creates, changeCreate)
}

// AddChangeUpdate adds a new update entry to the current object.
func (c *hetznerChanges) AddChangeUpdate(rrset *hcloud.ZoneRRSet, ttlOpts *hcloud.ZoneRRSetChangeTTLOpts, recordsOpts *hcloud.ZoneRRSetSetRecordsOpts, updateOpts *hcloud.ZoneRRSetUpdateOpts) {
	changeUpdate := &hetznerChangeUpdate{
		rrset:       rrset,
		ttlOpts:     ttlOpts,
		recordsOpts: recordsOpts,
		updateOpts:  updateOpts,
	}
	c.updates = append(c.updates, changeUpdate)
}

// AddChangeDelete adds a new delete entry to the current object.
func (c *hetznerChanges) AddChangeDelete(rrset *hcloud.ZoneRRSet) {
	changeDelete := &hetznerChangeDelete{
		rrset: rrset,
	}
	c.deletes = append(c.deletes, changeDelete)
}

// applyDeletes processes the records to be deleted.
func (c hetznerChanges) applyDeletes(ctx context.Context, client apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.deletes {
		log.WithFields(e.GetLogFields()).Debug("Deleting domain record")
		log.Infof("Deleting record [%s] from zone [%s]", e.rrset.Name, e.rrset.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, _, err := client.DeleteRRSet(ctx, e.rrset); err != nil {
			metrics.IncFailedApiCallsTotal(actDeleteRRSet)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actDeleteRRSet)
		metrics.AddApiDelayHist(actDeleteRRSet, delay.Milliseconds())
	}
	return nil
}

// applyCreates processes the records to be created.
func (c hetznerChanges) applyCreates(ctx context.Context, client apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.creates {
		zone := e.zone
		opts := e.opts
		if opts.TTL == nil {
			ttl := c.defaultTTL
			opts.TTL = &ttl
		}
		log.WithFields(e.GetLogFields()).Debug("Creating domain record")
		log.Infof("Creating record [%s] of type [%s] with records(%s) in zone [%s]",
			opts.Name, opts.Type, getRRSetRecordsString(opts.Records), zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, _, err := client.CreateRRSet(ctx, zone, opts); err != nil {
			metrics.IncFailedApiCallsTotal(actCreateRRSet)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actCreateRRSet)
		metrics.AddApiDelayHist(actCreateRRSet, delay.Milliseconds())
	}
	return nil
}

// applyUpdates processes the records to be updated.
func (c hetznerChanges) applyUpdates(ctx context.Context, client apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.updates {
		rrset := e.rrset
		recordOpts := e.recordsOpts
		ttlOpts := e.ttlOpts
		updateOpts := e.updateOpts
		log.WithFields(e.GetLogFields()).Debug("Updating domain record")
		if recordOpts != nil {
			log.Infof("Updating recordset for ID [%s], Name [%s], Type [%s] in zone [%s]: %s",
				rrset.ID, rrset.Name, rrset.Type, rrset.Zone.Name, getRRSetRecordsString(recordOpts.Records))
			if c.dryRun {
				continue
			}
			start := time.Now()
			if _, _, err := client.UpdateRRSetRecords(ctx, rrset, *recordOpts); err != nil {
				metrics.IncFailedApiCallsTotal(actUpdateRRSetRecords)
				return err
			}
			delay := time.Since(start)
			metrics.IncSuccessfulApiCallsTotal(actUpdateRRSetRecords)
			metrics.AddApiDelayHist(actUpdateRRSetRecords, delay.Milliseconds())
		}
		if ttlOpts != nil {
			if ttlOpts.TTL == nil {
				ttl := c.defaultTTL
				ttlOpts.TTL = &ttl
			}
			log.Infof("Updating TTL for ID [%s], Name [%s], Type [%s] in zone [%s]: %d",
				rrset.ID, rrset.Name, rrset.Type, rrset.Zone.Name, *ttlOpts.TTL)
			if c.dryRun {
				continue
			}
			start := time.Now()
			if _, _, err := client.UpdateRRSetTTL(ctx, rrset, *ttlOpts); err != nil {
				metrics.IncFailedApiCallsTotal(actUpdateRRSetTTL)
				return err
			}
			delay := time.Since(start)
			metrics.IncSuccessfulApiCallsTotal(actUpdateRRSetTTL)
			metrics.AddApiDelayHist(actUpdateRRSetTTL, delay.Milliseconds())
		}
		if updateOpts != nil {
			logLabels := formatLabels(updateOpts.Labels)
			log.Infof("Updating labels for ID [%s], Name [%s], Type [%s] in zone [%s]: %s",
				rrset.ID, rrset.Name, rrset.Type, rrset.Zone.Name, logLabels)
			if c.dryRun {
				continue
			}
			start := time.Now()
			if _, _, err := client.UpdateRRSetLabels(ctx, rrset, *updateOpts); err != nil {
				metrics.IncFailedApiCallsTotal(actUpdateRRSetTTL)
				return err
			}
			delay := time.Since(start)
			metrics.IncSuccessfulApiCallsTotal(actUpdateRRSetTTL)
			metrics.AddApiDelayHist(actUpdateRRSetTTL, delay.Milliseconds())
		}
	}
	return nil
}

// ApplyChanges applies the planned changes using dnsClient.
func (c hetznerChanges) ApplyChanges(ctx context.Context, dnsClient apiClient) error {
	// No changes = nothing to do.
	if c.empty() {
		log.Debug("No changes to be applied found.")
		return nil
	}
	// Process records to be deleted.
	if err := c.applyDeletes(ctx, dnsClient); err != nil {
		return err
	}
	// Process record creations.
	if err := c.applyCreates(ctx, dnsClient); err != nil {
		return err
	}
	// Process record updates.
	if err := c.applyUpdates(ctx, dnsClient); err != nil {
		return err
	}
	return nil
}
