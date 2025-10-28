/*
 * Connector - functions for reading zones and records from Hetzner DNS
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

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
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
	GetZones(ctx context.Context, opts hcloud.ZoneListOpts) ([]*hcloud.Zone, *hcloud.Response, error)
	// GetRRSets returns the RRSets for a given zone.
	GetRRSets(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetListOpts) ([]*hcloud.ZoneRRSet, *hcloud.Response, error)
	// CreateRRSet creates a new RRSet.
	CreateRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetAddRecordsOpts) (*hcloud.Action, *hcloud.Response, error)
	// UpdateRRSetTTL updates an RRSet's TTL.
	UpdateRRSetTTL(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetChangeTTLOpts) (*hcloud.Action, *hcloud.Response, error)
	// UpdateRRSetRecords updates the records of an RRSet.
	UpdateRRSetRecords(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetSetRecordsOpts) (*hcloud.Action, *hcloud.Response, error)
	// DeleteRRSet deletes an RRSet.
	DeleteRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet) (hcloud.ZoneRRSetDeleteResult, *hcloud.Response, error)
}

// fetchRecords fetches all records for a given zone.
func fetchRecords(ctx context.Context, zone *hcloud.Zone, client apiClient, batchSize int) ([]*hcloud.ZoneRRSet, error) {
	metrics := metrics.GetOpenMetricsInstance()
	records := []*hcloud.ZoneRRSet{}
	opts := hcloud.ZoneRRSetListOpts{
		ListOpts: hcloud.ListOpts{PerPage: batchSize},
	}

	for {
		start := time.Now()
		pagedRecords, resp, err := client.GetRRSets(ctx, zone, opts)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetRecords)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetRecords)
		metrics.AddApiDelayHist(actGetRecords, delay.Milliseconds())
		records = append(records, pagedRecords...)

		if resp == nil || resp.Meta.Pagination == nil || resp.Meta.Pagination.LastPage <= resp.Meta.Pagination.Page {
			break
		}

		opts.Page = resp.Meta.Pagination.Page + 1
	}

	return records, nil
}

// fetchZones fetches all the zones from the client.
func fetchZones(ctx context.Context, client apiClient, batchSize int) ([]*hcloud.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	zones := []*hcloud.Zone{}
	opts := hcloud.ZoneListOpts{
		ListOpts: hcloud.ListOpts{PerPage: batchSize},
	}

	for {
		start := time.Now()
		pagedZones, resp, err := client.GetZones(ctx, opts)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetZones)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetZones)
		metrics.AddApiDelayHist(actGetZones, delay.Milliseconds())
		zones = append(zones, pagedZones...)

		if resp == nil || resp.Meta.Pagination == nil || resp.Meta.Pagination.LastPage <= resp.Meta.Pagination.Page {
			break
		}

		opts.Page = resp.Meta.Pagination.Page + 1
	}

	return zones, nil
}
