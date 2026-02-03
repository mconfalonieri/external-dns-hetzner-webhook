/*
 * Connector - functions for reading zones and records from Hetzner DNS
 *
 * Copyright 2026 Marco Confalonieri.
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
package hetznerdns

import (
	"context"
	"time"

	"external-dns-hetzner-webhook/internal/metrics"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
)

const (
	actGetZones     = "get_zones"
	actGetRecords   = "get_records"
	actCreateRecord = "create_record"
	actUpdateRecord = "update_record"
	actDeleteRecord = "delete_record"
)

// apiClient is an abstraction of the REST API client.
type apiClient interface {
	// GetZones returns the available zones.
	GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error)
	// GetRecords returns the records for a given zone.
	GetRecords(ctx context.Context, opts hdns.RecordListOpts) ([]*hdns.Record, *hdns.Response, error)
	// CreateRecord creates a record.
	CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error)
	// UpdateRecord updates a single record.
	UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error)
	// DeleteRecord deletes a single record.
	DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error)
}

// fetchRecords fetches all records for a given zoneID.
func fetchRecords(ctx context.Context, zoneID string, dnsClient apiClient, batchSize int) ([]hdns.Record, error) {
	metrics := metrics.GetOpenMetricsInstance()
	records := []hdns.Record{}
	listOptions := &hdns.RecordListOpts{ListOpts: hdns.ListOpts{PerPage: batchSize}, ZoneID: zoneID}
	for {
		start := time.Now()
		pagedRecords, resp, err := dnsClient.GetRecords(ctx, *listOptions)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetRecords)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetRecords)
		metrics.AddApiDelayHist(actGetRecords, delay.Milliseconds())
		for _, r := range pagedRecords {
			records = append(records, *r)
		}

		if resp == nil || resp.Meta.Pagination == nil || resp.Meta.Pagination.LastPage <= resp.Meta.Pagination.Page {
			break
		}

		listOptions.Page = resp.Meta.Pagination.Page + 1
	}

	return records, nil
}

// fetchZones fetches all the zones from the DNS client.
func fetchZones(ctx context.Context, dnsClient apiClient, batchSize int) ([]hdns.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	zones := []hdns.Zone{}
	listOptions := &hdns.ZoneListOpts{ListOpts: hdns.ListOpts{PerPage: batchSize}}
	for {
		start := time.Now()
		pagedZones, resp, err := dnsClient.GetZones(ctx, *listOptions)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetZones)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetZones)
		metrics.AddApiDelayHist(actGetZones, delay.Milliseconds())
		for _, z := range pagedZones {
			zones = append(zones, *z)
		}

		if resp == nil || resp.Meta.Pagination == nil || resp.Meta.Pagination.LastPage <= resp.Meta.Pagination.Page {
			break
		}

		listOptions.Page = resp.Meta.Pagination.Page + 1
	}

	return zones, nil
}
