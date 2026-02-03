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

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// hetznerCloud is the cloud API client.
type hetznerCloud struct {
	client *hcloud.Client
}

// NewHetznerCloud returns a new client.
func NewHetznerCloud(apiKey string) (*hetznerCloud, error) {
	if apiKey == "" {
		return nil, errors.New("nil API key provided")
	}
	return &hetznerCloud{
		client: hcloud.NewClient(hcloud.WithToken(apiKey)),
	}, nil
}

// GetZones returns the available zones.
func (h hetznerCloud) GetZones(ctx context.Context, opts hcloud.ZoneListOpts) ([]*hcloud.Zone, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.List(ctx, opts)
}

// GetRRSets returns the RRSets for a given zone.
func (h hetznerCloud) GetRRSets(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetListOpts) ([]*hcloud.ZoneRRSet, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.ListRRSets(ctx, zone, opts)
}

// CreateRRSet creates a new RRSet.
func (h hetznerCloud) CreateRRSet(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts) (hcloud.ZoneRRSetCreateResult, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.CreateRRSet(ctx, zone, opts)
}

// UpdateRRSetTTL updates an RRSet's TTL.
func (h hetznerCloud) UpdateRRSetTTL(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetChangeTTLOpts) (*hcloud.Action, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.ChangeRRSetTTL(ctx, rrset, opts)
}

// UpdateRRSetRecords updates the records of an RRSet.
func (h hetznerCloud) UpdateRRSetRecords(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetSetRecordsOpts) (*hcloud.Action, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.SetRRSetRecords(ctx, rrset, opts)
}

// UpdateRRSetLabels updates the labels of an RRSet.
func (h hetznerCloud) UpdateRRSetLabels(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetUpdateOpts) (*hcloud.ZoneRRSet, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.UpdateRRSet(ctx, rrset, opts)
}

// DeleteRecord deletes an RRSet.
func (h hetznerCloud) DeleteRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet) (hcloud.ZoneRRSetDeleteResult, *hcloud.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.DeleteRRSet(ctx, rrset)
}
