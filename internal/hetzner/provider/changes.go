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
package provider

import (
	"context"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner/model"
	"external-dns-hetzner-webhook/internal/metrics"

	log "github.com/sirupsen/logrus"
)

// hetznerChange contains all changes to apply to DNS.
type hetznerChanges struct {
	dryRun     bool
	defaultTTL int

	creates []hetznerChangeCreate
	updates []hetznerChangeUpdate
	deletes []hetznerChangeDelete
}

// empty returns true if there are no changes left.
func (c *hetznerChanges) empty() bool {
	return len(c.creates) == 0 && len(c.updates) == 0 && len(c.deletes) == 0
}

// AddChangeCreate adds a new creation entry to the current object.
func (c *hetznerChanges) AddChangeCreate(record model.Record) {
	changeCreate := hetznerChangeCreate(record)
	c.creates = append(c.creates, changeCreate)
}

// AddChangeUpdate adds a new update entry to the current object.
func (c *hetznerChanges) AddChangeUpdate(record model.Record) {
	changeUpdate := hetznerChangeUpdate(record)
	c.updates = append(c.updates, changeUpdate)
}

// AddChangeDelete adds a new delete entry to the current object.
func (c *hetznerChanges) AddChangeDelete(record model.Record) {
	changeDelete := hetznerChangeDelete(record)
	c.deletes = append(c.deletes, changeDelete)
}

// applyDeletes processes the records to be deleted.
func (c hetznerChanges) applyDeletes(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.deletes {
		log.WithFields(e.GetLogFields()).Debug("Deleting domain record")
		log.Infof("Deleting record [%s] from zone [%s]", e.Name, e.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if err := dnsClient.DeleteRecord(ctx, e.ID); err != nil {
			metrics.IncFailedApiCallsTotal(actDeleteRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actDeleteRecord)
		metrics.AddApiDelayHist(actDeleteRecord, delay.Milliseconds())
	}
	return nil
}

// applyCreates processes the records to be created.
func (c hetznerChanges) applyCreates(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.creates {
		if e.TTL < 0 {
			e.TTL = c.defaultTTL
		}
		log.WithFields(e.GetLogFields()).Debug("Creating domain record")
		log.Infof("Creating record [%s] of type [%s] with value [%s] in zone [%s]",
			e.Name, e.Type, e.Value, e.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, err := dnsClient.CreateRecord(ctx, model.Record(e)); err != nil {
			metrics.IncFailedApiCallsTotal(actCreateRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actCreateRecord)
		metrics.AddApiDelayHist(actCreateRecord, delay.Milliseconds())
	}
	return nil
}

// applyUpdates processes the records to be updated.
func (c hetznerChanges) applyUpdates(ctx context.Context, dnsClient apiClient) error {
	metrics := metrics.GetOpenMetricsInstance()
	for _, e := range c.updates {
		if e.TTL < 0 {
			e.TTL = c.defaultTTL
		}
		log.WithFields(e.GetLogFields()).Debug("Updating domain record")
		log.Infof("Updating record ID [%s] with name [%s], type [%s], value [%s] and TTL [%d] in zone [%s]",
			e.ID, e.Name, e.Type, e.Value, e.TTL, e.Zone.Name)
		if c.dryRun {
			continue
		}
		start := time.Now()
		if _, err := dnsClient.UpdateRecord(ctx, e.ID, model.Record(e)); err != nil {
			metrics.IncFailedApiCallsTotal(actUpdateRecord)
			return err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actUpdateRecord)
		metrics.AddApiDelayHist(actUpdateRecord, delay.Milliseconds())
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
