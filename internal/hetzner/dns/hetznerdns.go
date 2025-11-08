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
package hetznerdns

import (
	"context"
	"errors"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
)

// hetznerDNS is the DNS client API.
type hetznerDNS struct {
	client *hdns.Client
}

// NewHetznerDNS returns a new client.
func NewHetznerDNS(apiKey string) (*hetznerDNS, error) {
	if apiKey == "" {
		return nil, errors.New("nil API key provided")
	}
	return &hetznerDNS{
		client: hdns.NewClient(hdns.WithToken(apiKey)),
	}, nil
}

// GetZones returns the available zones.
func (h hetznerDNS) GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.List(ctx, opts)
}

// GetRecords returns the records for a given zone.
func (h hetznerDNS) GetRecords(ctx context.Context, opts hdns.RecordListOpts,
) ([]*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.List(ctx, opts)
}

// CreateRecord creates a record.
func (h hetznerDNS) CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Create(ctx, opts)
}

// UpdateRecord updates a single record.
func (h hetznerDNS) UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Update(ctx, record, opts)
}

// DeleteRecord deletes a single record.
func (h hetznerDNS) DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Delete(ctx, record)
}
