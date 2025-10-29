/*
 * HetznerDNS - Common test routines for the hetzner package.
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
package hetznerdns

import (
	"context"
	"testing"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"
)

// testTTL is a test ttl.
var testTTL = 7200

// zonesResponse simulates a response that returns a list of zones.
type zonesResponse struct {
	zones []*hdns.Zone
	resp  *hdns.Response
	err   error
}

// recordsResponse simulates a response that returns a list of records.
type recordsResponse struct {
	records []*hdns.Record
	resp    *hdns.Response
	err     error
}

// recordResponse simulates a response that returns a single record.
type recordResponse struct {
	record *hdns.Record
	resp   *hdns.Response
	err    error
}

// deleteResponse simulates a response to a record deletion request.
type deleteResponse struct {
	resp *hdns.Response
	err  error
}

// mockClientState keeps track of which methods were called.
type mockClientState struct {
	GetZonesCalled     bool
	GetRecordsCalled   bool
	CreateRecordCalled bool
	UpdateRecordCalled bool
	DeleteRecordCalled bool
}

// mockClient represents the mock client used to simulate calls to the DNS API.
type mockClient struct {
	getZones            zonesResponse
	getRecords          recordsResponse
	createRecord        recordResponse
	updateRecord        recordResponse
	deleteRecord        deleteResponse
	filterRecordsByZone bool
	state               mockClientState
}

// GetState returns the internal state
func (m mockClient) GetState() mockClientState {
	return m.state
}

// GetZones simulates a request to get a list of zones.
func (m *mockClient) GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error) {
	r := m.getZones
	m.state.GetZonesCalled = true
	return r.zones, r.resp, r.err
}

// filterRecordsByZone filters the records, returning only those for the selected zone.
func filterRecordsByZone(r recordsResponse, opts hdns.RecordListOpts) []*hdns.Record {
	records := make([]*hdns.Record, 0)
	for _, rec := range r.records {
		if rec != nil && rec.Zone.ID == opts.ZoneID {
			records = append(records, rec)
		}
	}
	return records
}

// GetRecords simulates a request to get a list of records for a given zone.
func (m *mockClient) GetRecords(ctx context.Context, opts hdns.RecordListOpts) ([]*hdns.Record, *hdns.Response, error) {
	r := m.getRecords
	m.state.GetRecordsCalled = true
	var records []*hdns.Record
	if m.filterRecordsByZone {
		records = filterRecordsByZone(r, opts)
		r.resp.Meta.Pagination.TotalEntries = len(records) // "smart" handling
	} else {
		records = r.records
	}
	return records, r.resp, r.err
}

// CreateRecord simulates a request to create a DNS record.
func (m *mockClient) CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.createRecord
	m.state.CreateRecordCalled = true
	return r.record, r.resp, r.err
}

// UpdateRecord simulates a request to update a DNS record.
func (m *mockClient) UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.updateRecord
	m.state.UpdateRecordCalled = true
	return r.record, r.resp, r.err
}

// DeleteRecord simulates a request to delete a DNS record.
func (m *mockClient) DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error) {
	r := m.deleteRecord
	m.state.DeleteRecordCalled = true
	return r.resp, r.err
}

// assertError checks if an error is thrown when expected.
func assertError(t *testing.T, expected, actual error) bool {
	var expError bool
	if expected == nil {
		assert.Nil(t, actual)
		expError = false
	} else {
		assert.EqualError(t, actual, expected.Error())
		expError = true
	}
	return expError
}
