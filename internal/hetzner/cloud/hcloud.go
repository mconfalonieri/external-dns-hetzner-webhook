/*
 * HCloud - This handles API calls using the hcloud library.
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
package cloud

import (
	"context"
	"errors"
	"external-dns-hetzner-webhook/internal/hetzner/model"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// hCloud is the Hetzner Cloud API client.
type hCloud struct {
	client *hcloud.Client
}

// NewHCloud returns a new client.
func NewHCloud(opts ...hcloud.ClientOption) *hCloud {
	return &hCloud{
		client: hcloud.NewClient(opts...),
	}
}

// GetZones returns the available zones.
func (h hCloud) GetZones(ctx context.Context, opts model.ZoneListOpts) ([]model.Zone, *model.Pagination, error) {
	zones, err := h.client.Zone.All(ctx)
	return getPZoneArray(zones), nil, err
}

// GetRecords returns the records for a given zone.
func (h hCloud)	GetRecords(ctx context.Context,	opts model.RecordListOpts) ([]model.Record,	*model.Pagination,	error) {
	zone, convErr := getHZoneFromID(opts.ZoneID)
	if convErr != nil {
		return nil, nil, errors.New("cannot read a ZoneID while reading records")
	}
	records, err := h.client.Zone.AllRRSets(ctx, zone)

}
	// CreateRecord creates a record.
	CreateRecord(ctx context.Context, record model.Record) (model.Record, *http.Response, error)
	// UpdateRecord updates a single record.
	UpdateRecord(ctx context.Context, id string, record model.Record) (model.Record, *http.Response, error)
	// DeleteRecord deletes a single record.
	DeleteRecord(ctx context.Context, id string) (*http.Response, error)
}
*/
