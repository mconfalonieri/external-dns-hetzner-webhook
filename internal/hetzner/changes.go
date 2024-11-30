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
package hetzner

import (
	"context"
	"time"

	"external-dns-hetzner-webhook/internal/metrics"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
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
func (c *hetznerChanges) AddChangeCreate(zoneID string, options *hdns.RecordCreateOpts) {
	changeCreate := &hetznerChangeCreate{
		ZoneID:  zoneID,
		Options: options,
	}
	c.creates = append(c.creates, changeCreate)
}

// AddChangeUpdate adds a new update entry to the current object.
func (c *hetznerChanges) AddChangeUpdate(zoneID string, record hdns.Record, options *hdns.RecordUpdateOpts) {
	changeUpdate := &hetznerChangeUpdate{
		ZoneID:  zoneID,
		Record:  record,
		Options: options,
	}
	c.updates = append(c.updates, changeUpdate)
}

// AddChangeDelete adds a new delete entry to the current object.
func (c *hetznerChanges) AddChangeDelete(zoneID string, record hdns.Record) {
	changeDelete := &hetznerChangeDelete{
		ZoneID: zoneID,
		Record: record,
	}
	c.deletes = append(c.deletes, changeDelete)
}

// applyDeletes processes the records to be deleted.
func (c hetznerChanges) applyDeletes(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.deletes {
		log.WithFields(e.GetLogFields()).Debug("Deleting domain record")
		log.Infof("Deleting record [%s] from zone [%s]", e.Record.Name, e.Record.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, err := dnsClient.DeleteRecord(ctx, &e.Record); err != nil {
			metrics.IncFailedApiCallsTotal(actDeleteRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actDeleteRecord)
		metrics.AddApiDelayCount(actDeleteRecord, delay.Milliseconds())
	}
	return nil
}

// applyCreates processes the records to be created.
func (c hetznerChanges) applyCreates(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.creates {
		opt := e.Options
		if opt.Ttl == nil {
			ttl := c.defaultTTL
			opt.Ttl = &ttl
		}
		log.WithFields(e.GetLogFields()).Debug("Creating domain record")
		log.Infof("Creating record [%s] of type [%s] with value [%s] in zone [%s]",
			opt.Name, opt.Type, opt.Value, opt.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, _, err := dnsClient.CreateRecord(ctx, *opt); err != nil {
			metrics.IncFailedApiCallsTotal(actCreateRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actCreateRecord)
		metrics.AddApiDelayCount(actCreateRecord, delay.Milliseconds())
	}
	return nil
}

// applyUpdates processes the records to be updated.
func (c hetznerChanges) applyUpdates(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.updates {
		opt := e.Options
		if opt.Ttl == nil {
			ttl := c.defaultTTL
			opt.Ttl = &ttl
		}
		log.WithFields(e.GetLogFields()).Debug("Updating domain record")
		log.Infof("Updating record ID [%s] with name [%s], type [%s], value [%s] and TTL [%d] in zone [%s]",
			e.Record.ID, opt.Name, opt.Type, opt.Value, *opt.Ttl, opt.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, _, err := dnsClient.UpdateRecord(ctx, &e.Record, *opt); err != nil {
			metrics.IncFailedApiCallsTotal(actUpdateRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actUpdateRecord)
		metrics.AddApiDelayCount(actUpdateRecord, delay.Milliseconds())
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
