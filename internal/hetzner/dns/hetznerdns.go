/*
 * HetznerDNS - This handles API calls towards Hetzner DNS.
 *
 * Copyright 2023 Marco Confalonieri.
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
package dns

import (
	"context"

	"external-dns-hetzner-webhook/internal/hetzner/model"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
)

// hetznerDNS is the DNS client API.
type hetznerDNS struct {
	client *hdns.Client
}

// NewHetznerDNS returns a new client.
func NewHetznerDNS(apiKey string) *hetznerDNS {
	return &hetznerDNS{
		client: hdns.NewClient(hdns.WithToken(apiKey)),
	}
}

// GetZones returns the available zones.
func (h hetznerDNS) GetZones(ctx context.Context, opts model.ZoneListOpts) ([]model.Zone, *model.Pagination, error) {
	zoneClient := h.client.Zone
	libOpts := getDNSZoneListOpts(opts)
	libZones, libResponse, err := zoneClient.List(ctx, libOpts)
	return getPZoneArray(libZones), getPaginationMeta(libResponse.Meta), err
}

// GetRecords returns the records for a given zone.
func (h hetznerDNS) GetRecords(ctx context.Context, opts model.RecordListOpts) ([]model.Record, *model.Pagination, error) {
	recordClient := h.client.Record
	libOpts := getDNSRecordListOpts(opts)
	libRecords, libResponse, err := recordClient.List(ctx, libOpts)
	return getPRecordArray(libRecords), getPaginationMeta(libResponse.Meta), err
}

// CreateRecord creates a record.
func (h hetznerDNS) CreateRecord(ctx context.Context, record model.Record) (model.Record, error) {
	recordClient := h.client.Record
	libOpts := getDNSRecordCreateOpts(record)
	libRecord, _, err := recordClient.Create(ctx, libOpts)
	return getRecord(*libRecord), err
}

// UpdateRecord updates a single record.
func (h hetznerDNS) UpdateRecord(ctx context.Context, id string, record model.Record) (model.Record, error) {
	recordClient := h.client.Record
	libOpts := getDNSRecordUpdateOpts(record)
	libRecord, _, err := recordClient.Update(ctx, &hdns.Record{ID: id}, libOpts)
	return getRecord(*libRecord), err
}

// DeleteRecord deletes a single record.
func (h hetznerDNS) DeleteRecord(ctx context.Context, id string) error {
	recordClient := h.client.Record
	_, err := recordClient.Delete(ctx, &hdns.Record{ID: id})
	return err
}
