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
package provider

import (
	"context"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner/model"
	"external-dns-hetzner-webhook/internal/metrics"
)

const (
	actGetZones     = "get_zones"
	actGetRecords   = "get_records"
	actCreateRecord = "create_record"
	actUpdateRecord = "update_record"
	actDeleteRecord = "delete_record"
)

// fetchRecords fetches all records for a given zoneID.
func fetchRecords(ctx context.Context, zoneID string, dnsClient apiClient, batchSize int) ([]model.Record, error) {
	metrics := metrics.GetOpenMetricsInstance()
	records := []model.Record{}
	listOpts := model.RecordListOpts{
		ListOpts: model.ListOpts{ItemsPerPage: batchSize},
		ZoneID:   zoneID,
	}
	for {
		start := time.Now()
		pagedRecords, resp, pagination, err := dnsClient.GetRecords(ctx, listOpts)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetRecords)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetRecords)
		metrics.AddApiDelayHist(actGetRecords, delay.Milliseconds())
		records = append(records, pagedRecords...)

		if resp == nil || pagination == nil || pagination.LastPage <= pagination.PageIdx {
			break
		}

		listOpts.ListOpts.PageIdx = pagination.PageIdx + 1
	}

	return records, nil
}

// fetchZones fetches all the zones from the DNS client.
func fetchZones(ctx context.Context, dnsClient apiClient, batchSize int) ([]model.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	zones := []model.Zone{}
	listOpts := &model.ZoneListOpts{
		ListOpts: model.ListOpts{ItemsPerPage: batchSize},
	}

	for {
		start := time.Now()
		pagedZones, resp, pagination, err := dnsClient.GetZones(ctx, *listOpts)
		if err != nil {
			metrics.IncFailedApiCallsTotal(actGetZones)
			return nil, err
		}
		delay := time.Since(start)
		metrics.IncSuccessfulApiCallsTotal(actGetZones)
		metrics.AddApiDelayHist(actGetZones, delay.Milliseconds())
		zones = append(zones, pagedZones...)

		if resp == nil || pagination == nil || pagination.LastPage <= pagination.PageIdx {
			break
		}

		listOpts.PageIdx = pagination.PageIdx + 1
	}

	return zones, nil
}
