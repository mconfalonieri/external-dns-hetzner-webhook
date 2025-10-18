/*
 * API client interface
 *
 * Copyright 2025 Marco Confalonieri.
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
	"net/http"

	"external-dns-hetzner-webhook/internal/hetzner/model"
)

// apiClient is an abstraction of the REST API client.
type apiClient interface {
	// GetZones returns the available zones.
	GetZones(
		ctx context.Context,
		opts model.ZoneListOpts,
	) (
		[]model.Zone,
		*http.Response,
		*model.Pagination,
		error,
	)
	// GetRecords returns the records for a given zone.
	GetRecords(
		ctx context.Context,
		opts model.RecordListOpts,
	) (
		[]model.Record,
		*http.Response,
		*model.Pagination,
		error,
	)
	// CreateRecord creates a record.
	CreateRecord(ctx context.Context, record model.Record) (model.Record, *http.Response, error)
	// UpdateRecord updates a single record.
	UpdateRecord(ctx context.Context, id string, record model.Record) (model.Record, *http.Response, error)
	// DeleteRecord deletes a single record.
	DeleteRecord(ctx context.Context, id string) (*http.Response, error)
}
