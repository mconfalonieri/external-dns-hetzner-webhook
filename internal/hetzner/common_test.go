/*
 * Common test routines for the hetzner package.
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
	"testing"
	"time"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	hschema "github.com/jobstoit/hetzner-dns-go/dns/schema"
	"sigs.k8s.io/external-dns/endpoint"
)

// testTime is a time used in records.
var testTime = time.Date(2021, 8, 15, 14, 30, 45, 100, time.Local)

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
	getZones     zonesResponse
	getRecords   recordsResponse
	createRecord recordResponse
	updateRecord recordResponse
	deleteRecord deleteResponse
	adjustZone   bool
	state        mockClientState
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

// adjustZone adjusts the records for the selected zone.
func adjustZone(r *recordsResponse, opts hdns.RecordListOpts) {
	for idx, rec := range r.records {
		if rec != nil {
			rec.Zone.ID = opts.ZoneID
		}
		r.records[idx] = rec
	}
}

// GetRecords simulates a request to get a list of records for a given zone.
func (m *mockClient) GetRecords(ctx context.Context, opts hdns.RecordListOpts) ([]*hdns.Record, *hdns.Response, error) {
	r := m.getRecords
	m.state.GetRecordsCalled = true
	if m.adjustZone {
		adjustZone(&r, opts)
	}
	return r.records, r.resp, r.err
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

// unpointedZones transforms a slice of zone pointers into a slice of zones.
func unpointedZones(zones []*hdns.Zone) []hdns.Zone {
	ret := make([]hdns.Zone, len(zones))
	for i, z := range zones {
		if z == nil {
			continue
		}
		ret[i] = *z
	}
	return ret
}

// unpointedRecords transforms a slice of record pointers into a slice of
// records.
func unpointedRecords(records []*hdns.Record) []hdns.Record {
	ret := make([]hdns.Record, len(records))
	for i, r := range records {
		if r == nil {
			continue
		}
		ret[i] = *r
	}
	return ret
}

// checkError checks if an error is thrown when expected.
func checkError(t *testing.T, err error, errExp bool) {
	isErr := (err != nil)
	if (isErr && !errExp) || (!isErr && errExp) {
		t.Fail()
	}
}

// toEndpoints transforms a slice of records in a slice of endpoint pointers.
func toEndpoints(records []hdns.Record) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, 0, len(records))
	for _, r := range records {
		e := endpoint.Endpoint{
			DNSName:       r.Name,
			SetIdentifier: r.ID,
			Targets:       []string{r.Value},
			RecordTTL:     endpoint.TTL(r.Ttl),
			RecordType:    string(r.Type),
		}
		endpoints = append(endpoints, &e)
	}
	return endpoints
}

// buildTestRecord builds a record according to parameters. The indexes of
// the params argument must contain:
// - 0: the record type
// - 1: the record name
// - 2: the record value
func buildTestRecord(params [3]string, zoneId string) *hdns.Record {
	return &hdns.Record{
		Type:     hdns.RecordType(params[0]),
		ID:       "id_" + params[1],
		Name:     params[1],
		Value:    params[2],
		Zone:     &hdns.Zone{ID: zoneId},
		Created:  hschema.HdnsTime(testTime),
		Modified: hschema.HdnsTime(testTime),
		Ttl:      7200,
	}
}

// buildTestRecords builds some test records for the given zoneId.
func buildTestRecords(zoneId string) []*hdns.Record {
	fixture := [][3]string{
		{"A", "www", "127.0.0.1"},
		{"MX", "mail", "127.0.0.1"},
		{"CNAME", "ftp", "www.alpha.com"},
	}
	records := make([]*hdns.Record, len(fixture))
	for i, f := range fixture {
		rec := buildTestRecord(f, zoneId)
		records[i] = rec
	}
	return records
}

// buildTestZones bulids some test zones.
func buildTestZones() []*hdns.Zone {
	zones := []*hdns.Zone{
		{
			ID:       "zoneIDAlpha",
			Created:  hschema.HdnsTime(testTime),
			Modified: hschema.HdnsTime(testTime),
			Name:     "alpha.com",
			Ttl:      7200,
		},
		{
			ID:       "zoneIDBeta",
			Created:  hschema.HdnsTime(testTime),
			Modified: hschema.HdnsTime(testTime),
			Name:     "beta.com",
			Ttl:      7200,
		},
	}
	return zones
}
