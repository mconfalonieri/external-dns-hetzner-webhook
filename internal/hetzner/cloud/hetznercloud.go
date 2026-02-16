/*
 * HetznerCloud - This handles API calls towards Hetzner Cloud DNS API.
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
package hetznercloud

import (
	"context"
	"errors"
	"time"

	"external-dns-hetzner-webhook/internal/metrics"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	// Action constants used for metrics.
	actGetZones           = "get_zones"
	actGetRRSets          = "get_rrsets"
	actCreateRRSet        = "create_rrset"
	actUpdateRRSetTTL     = "update_rrset_ttl"
	actUpdateRRSetRecords = "update_rrset_records"
	actUpdateRRSet        = "update_rrset"
	actDeleteRRSet        = "delete_rrset"
	actExportZonefile     = "export_zonefile"
	actImportZonefile     = "import_zonefile"
)

// hetznerCloud is the Cloud API client.
type hetznerCloud struct {
	client  *hcloud.Client
	metrics *metrics.OpenMetrics
}

// NewHetznerCloud returns a new client. The API key is passed as an argument.
func NewHetznerCloud(apiKey string) (*hetznerCloud, error) {
	if apiKey == "" {
		return nil, errors.New("nil API key provided")
	}
	return &hetznerCloud{
		client:  hcloud.NewClient(hcloud.WithToken(apiKey)),
		metrics: metrics.GetOpenMetricsInstance(),
	}, nil
}

// GetZones returns the available zones.
func (h hetznerCloud) GetZones(ctx context.Context, opts hcloud.ZoneListOpts) ([]*hcloud.Zone, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.List(ctx, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actGetZones)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actGetZones)
	h.metrics.AddApiDelayHist(actGetZones, delay.Milliseconds())
	return result, response, err
}

// GetRRSets returns the recordset found in a given zone.
func (h hetznerCloud) GetRRSets(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetListOpts) ([]*hcloud.ZoneRRSet, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.ListRRSets(ctx, zone, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actGetRRSets)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actGetRRSets)
	h.metrics.AddApiDelayHist(actGetRRSets, delay.Milliseconds())
	return result, response, err
}

// CreateRRSet creates a new recordset in the specified zone.
func (h hetznerCloud) CreateRRSet(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts) (hcloud.ZoneRRSetCreateResult, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.CreateRRSet(ctx, zone, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actCreateRRSet)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actCreateRRSet)
	h.metrics.AddApiDelayHist(actCreateRRSet, delay.Milliseconds())
	return result, response, err
}

// UpdateRRSetTTL updates the TTL of a recordset. It is not possible to set a
// different TTL for each target.
func (h hetznerCloud) UpdateRRSetTTL(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetChangeTTLOpts) (*hcloud.Action, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.ChangeRRSetTTL(ctx, rrset, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actCreateRRSet)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actCreateRRSet)
	h.metrics.AddApiDelayHist(actCreateRRSet, delay.Milliseconds())
	return result, response, err
}

// UpdateRRSetRecords updates the targets of a recordset. The provided targets
// overwrite completely the previous ones.
func (h hetznerCloud) UpdateRRSetRecords(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetSetRecordsOpts) (*hcloud.Action, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.SetRRSetRecords(ctx, rrset, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actUpdateRRSetRecords)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actUpdateRRSetRecords)
	h.metrics.AddApiDelayHist(actUpdateRRSetRecords, delay.Milliseconds())
	return result, response, err
}

// UpdateRRSetLabels updates the labels of a recordset.
func (h hetznerCloud) UpdateRRSetLabels(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetUpdateOpts) (*hcloud.ZoneRRSet, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.UpdateRRSet(ctx, rrset, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actUpdateRRSet)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actUpdateRRSet)
	h.metrics.AddApiDelayHist(actUpdateRRSet, delay.Milliseconds())
	return result, response, err
}

// DeleteRecord deletes a recordset from a zone.
func (h hetznerCloud) DeleteRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet) (hcloud.ZoneRRSetDeleteResult, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.DeleteRRSet(ctx, rrset)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actDeleteRRSet)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actDeleteRRSet)
	h.metrics.AddApiDelayHist(actDeleteRRSet, delay.Milliseconds())
	return result, response, err
}

// ExportZonefile downloads a zonefile from Hetzner.
func (h hetznerCloud) ExportZonefile(ctx context.Context, zone *hcloud.Zone) (hcloud.ZoneExportZonefileResult, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.ExportZonefile(ctx, zone)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actExportZonefile)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actExportZonefile)
	h.metrics.AddApiDelayHist(actExportZonefile, delay.Milliseconds())
	return result, response, err
}

// ImportZonefile uploads a zonefile to Hetzner.
func (h hetznerCloud) ImportZonefile(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneImportZonefileOpts) (*hcloud.Action, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	start := time.Now()
	result, response, err := zoneClient.ImportZonefile(ctx, zone, opts)
	if err != nil {
		h.metrics.IncFailedApiCallsTotal(actImportZonefile)
	}
	delay := time.Since(start)
	h.metrics.IncSuccessfulApiCallsTotal(actImportZonefile)
	h.metrics.AddApiDelayHist(actImportZonefile, delay.Milliseconds())
	return result, response, err
}
