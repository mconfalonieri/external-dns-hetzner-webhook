/*
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
package hetzner

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	hschema "github.com/jobstoit/hetzner-dns-go/dns/schema"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

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

// mockClient represents the mock client used to simulate calls to the DNS API.
type mockClient struct {
	getZones     zonesResponse
	getRecords   recordsResponse
	createRecord recordResponse
	updateRecord recordResponse
	deleteRecord deleteResponse
	adjustZone   bool
}

// GetZones simulates a request to get a list of zones.
func (m *mockClient) GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error) {
	r := m.getZones
	return r.zones, r.resp, r.err
}

// GetRecords simulates a request to get a list of records for a given zone.
func (m *mockClient) GetRecords(ctx context.Context, opts hdns.RecordListOpts) ([]*hdns.Record, *hdns.Response, error) {
	r := m.getRecords
	if m.adjustZone {
		for idx, rec := range r.records {
			if rec != nil {
				rec.Zone.ID = opts.ZoneID
			}
			r.records[idx] = rec
		}
	}
	return r.records, r.resp, r.err
}

// CreateRecord simulates a request to create a DNS record.
func (m *mockClient) CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.createRecord
	return r.record, r.resp, r.err
}

// UpdateRecord simulates a request to update a DNS record.
func (m *mockClient) UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.updateRecord
	return r.record, r.resp, r.err
}

// DeleteRecord simulates a request to delete a DNS record.
func (m *mockClient) DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error) {
	r := m.deleteRecord
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

// test time used in records.
var testTime = time.Date(2021, 8, 15, 14, 30, 45, 100, time.Local)

// buildTestZones bulids some test zones.
func buildTestZones() []*hdns.Zone {
	zones := []*hdns.Zone{
		{
			ID:       "id_a",
			Created:  hschema.HdnsTime(testTime),
			Modified: hschema.HdnsTime(testTime),
			Name:     "a.com",
			Ttl:      7200,
		},
		{
			ID:       "id_b",
			Created:  hschema.HdnsTime(testTime),
			Modified: hschema.HdnsTime(testTime),
			Name:     "b.com",
			Ttl:      7200,
		},
	}
	return zones
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
		{"CNAME", "ftp", "www.a.com"},
	}
	records := make([]*hdns.Record, len(fixture))
	for i, f := range fixture {
		rec := buildTestRecord(f, zoneId)
		records[i] = rec
	}
	return records
}

// Test_NewHetznerProvider tests NewHetznerProvider().
func Test_NewHetznerProvider(t *testing.T) {
	cfg := Configuration{
		APIKey:       "testKey",
		DryRun:       true,
		Debug:        true,
		BatchSize:    50,
		DefaultTTL:   3600,
		DomainFilter: []string{"a.com, b.com"},
	}

	p, _ := NewHetznerProvider(&cfg)

	assert.Equal(t, cfg.DryRun, p.dryRun)
	assert.Equal(t, cfg.Debug, p.debug)
	assert.Equal(t, cfg.BatchSize, p.batchSize)
	assert.Equal(t, cfg.DefaultTTL, p.defaultTTL)
	actualJSON, _ := p.domainFilter.MarshalJSON()
	expectedJSON, _ := GetDomainFilter(cfg).MarshalJSON()
	assert.Equal(t, actualJSON, expectedJSON)
}

// Test_hetznerChanges_Empty tests hetznerChanges.Empty().
func Test_hetznerChanges_Empty(t *testing.T) {
	type testCase struct {
		name     string
		changes  hetznerChanges
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.changes.Empty()
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "Empty",
			changes:  hetznerChanges{},
			expected: true,
		},
		{
			name: "Creations",
			changes: hetznerChanges{
				Creates: []*hetznerChangeCreate{
					{
						Domain:  "a.com",
						Options: &hdns.RecordCreateOpts{},
					},
				},
			},
		},
		{
			name: "Updates",
			changes: hetznerChanges{
				Updates: []*hetznerChangeUpdate{
					{
						Domain:       "a.com",
						DomainRecord: hdns.Record{},
						Options:      &hdns.RecordUpdateOpts{},
					},
				},
			},
		},
		{
			name: "Deletions",
			changes: hetznerChanges{
				Deletes: []*hetznerChangeDelete{
					{
						Domain:   "a.com",
						RecordID: "testId",
					},
				},
			},
		},
		{
			name: "All",
			changes: hetznerChanges{
				Creates: []*hetznerChangeCreate{
					{
						Domain:  "a.com",
						Options: &hdns.RecordCreateOpts{},
					},
				},
				Updates: []*hetznerChangeUpdate{
					{
						Domain:       "a.com",
						DomainRecord: hdns.Record{},
						Options:      &hdns.RecordUpdateOpts{},
					},
				},
				Deletes: []*hetznerChangeDelete{
					{
						Domain:   "a.com",
						RecordID: "testId",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_Zones tests HetznerProvider.Zones().
func Test_Zones(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			zones []hdns.Zone
			err   bool
		}
	}

	testZones := buildTestZones()

	run := func(t *testing.T, tc testCase) {
		resp, err := tc.provider.Zones(context.Background())
		checkError(t, err, tc.expected.err)
		if !tc.expected.err {
			assert.Equal(t, reflect.DeepEqual(resp, tc.expected.zones), true)
		}
	}

	testCases := []testCase{
		{
			name: "Zones returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: testZones,
						resp: &hdns.Response{
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testZones),
								},
							},
						},
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   bool
			}{
				zones: unpointedZones(testZones),
			},
		},
		{
			name: "Error returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_AdjustEndpoints tests HetznerProvider.AdjustEndpoints().
func Test_AdjustEndpoints(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		input    []*endpoint.Endpoint
		expected []*endpoint.Endpoint
	}

	endpoints := toEndpoints(unpointedRecords(buildTestRecords("id_a")))

	run := func(t *testing.T, tc testCase) {
		actual, _ := tc.provider.AdjustEndpoints(tc.input)
		assert.Equal(t, len(actual), len(tc.expected))
	}

	testCases := []testCase{
		{
			name:     "Empty list",
			input:    []*endpoint.Endpoint{},
			expected: []*endpoint.Endpoint{},
		},
		{
			name:     "Some elements",
			input:    endpoints,
			expected: endpoints,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_mergeEndpointsByNameType tests
// hetznerProvider.mergeEndpointsByNameType().
func Test_mergeEndpointsByNameType(t *testing.T) {
	mkEndpoint := func(params [3]string) *endpoint.Endpoint {
		return &endpoint.Endpoint{
			RecordType: params[0],
			DNSName:    params[1],
			Targets:    []string{params[2]},
		}
	}

	type testCase struct {
		name        string
		input       [][3]string
		expectedLen int
	}

	run := func(t *testing.T, tc testCase) {
		input := make([]*endpoint.Endpoint, 0, len(tc.input))
		for _, r := range tc.input {
			input = append(input, mkEndpoint(r))
		}
		actual := mergeEndpointsByNameType(input)
		assert.Equal(t, len(actual), tc.expectedLen)
	}

	testCases := []testCase{
		{
			name: "1:1 endpoint",
			input: [][3]string{
				{"A", "www.alfa.com", "8.8.8.8"},
				{"A", "www.beta.com", "9.9.9.9"},
				{"A", "www.gamma.com", "1.1.1.1"},
			},
			expectedLen: 3,
		},
		{
			name: "6:4 endpoint",
			input: [][3]string{
				{"A", "www.alfa.com", "1.1.1.1"},
				{"A", "www.beta.com", "2.2.2.2"},
				{"A", "www.beta.com", "3.3.3.3"},
				{"A", "www.gamma.com", "4.4.4.4"},
				{"A", "www.gamma.com", "5.5.5.5"},
				{"A", "www.delta.com", "6.6.6.6"},
			},
			expectedLen: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_Records tests HetznerProvider.Records().
func Test_Records(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			endpoints int
			err       bool
		}
	}

	testZones := buildTestZones()
	testRecords := buildTestRecords("id_a")

	run := func(t *testing.T, tc testCase) {
		actual, err := tc.provider.Records(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			assert.Equal(t, len(actual), tc.expected.endpoints)
		}
	}

	testCases := []testCase{
		{
			name: "Records returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: testZones,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testZones),
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: testRecords,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
						},
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 4, // MX test records will not show up
			},
		},
		{
			name: "Error in zones",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test zones error"),
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_fetchRecords tests HetznerProvider.fetchRecords().
func Test_fetchRecords(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		input    string
		expected struct {
			endpoints int
			err       bool
		}
	}

	testRecords := buildTestRecords("id_a")

	run := func(t *testing.T, tc testCase) {
		actual, err := tc.provider.fetchRecords(context.Background(), tc.input)
		checkError(t, err, tc.expected.err)
		if err != nil {
			assert.Equal(t, len(actual), tc.expected.endpoints)
		}
	}

	testCases := []testCase{
		{
			name: "All records",
			provider: HetznerProvider{
				client: &mockClient{
					getRecords: recordsResponse{
						records: testRecords,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
						},
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 4, // MX test records will not show up
			},
		},
		{
			name: "Error",
			provider: HetznerProvider{
				client: &mockClient{
					getRecords: recordsResponse{
						err: errors.New("records test error"),
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_fetchZones tests HetznerProvider.fetchZones().
func Test_fetchZones(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			zones []hdns.Zone
			err   bool
		}
	}

	testZones := buildTestZones()

	run := func(t *testing.T, tc testCase) {
		actual, err := tc.provider.fetchZones(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			assert.ElementsMatch(t, actual, tc.expected.zones)
		}
	}

	testCases := []testCase{
		{
			name: "Zones returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: testZones,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testZones),
								},
							},
						},
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   bool
			}{
				zones: unpointedZones(testZones), // 2 zones returned
			},
		},
		{
			name: "Error returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("zones test error"),
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_ensureZoneIDMappingPresent tests
// HetznerProvider.ensureZoneIDMappingPresent().
func Test_ensureZoneIDMappingPresent(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		input    []hdns.Zone
		expected map[string]string
	}

	run := func(t *testing.T, tc testCase) {
		tc.provider.ensureZoneIDMappingPresent(tc.input)
		actual := tc.provider.zoneIDNameMapper
		assert.EqualValues(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "empty zones",
			provider: HetznerProvider{},
			input:    []hdns.Zone{},
			expected: map[string]string{},
		},
		{
			name:     "some zones",
			provider: HetznerProvider{},
			input: []hdns.Zone{
				{
					ID:   "id_zone1",
					Name: "zone1",
				},
				{
					ID:   "id_zone2",
					Name: "zone2",
				},
			},
			expected: map[string]string{
				"id_zone1": "zone1",
				"id_zone2": "zone2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getRecordsByZoneID tests HetznerProvider.getRecordsByZoneID()
func Test_getRecordsByZoneID(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			records map[string][]hdns.Record
			zones   provider.ZoneIDName
			err     bool
		}
	}

	testZones := buildTestZones()
	testRecords := buildTestRecords("id_a")

	run := func(t *testing.T, tc testCase) {
		actualRecords, actualZones, err := tc.provider.getRecordsByZoneID(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			assert.Equal(t, len(actualRecords["id_a"]), len(testRecords))
			assert.Equal(t, len(actualRecords["id_b"]), len(testRecords))
			assert.EqualValues(t, actualZones, tc.expected.zones)
		}
	}

	testCases := []testCase{
		{
			name: "Zones returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: testZones,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testZones),
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: testRecords,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testRecords),
								},
							},
						},
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				records map[string][]hdns.Record
				zones   provider.ZoneIDName
				err     bool
			}{
				records: map[string][]hdns.Record{
					"id_a": unpointedRecords(buildTestRecords("id_a")),
					"id_b": unpointedRecords(buildTestRecords("id_b")),
				},
				zones: provider.ZoneIDName{
					"id_a": "a.com",
					"id_b": "b.com",
				}, // 2 zones returned
			},
		},
		{
			name: "Zone error returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test zone error"),
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				records map[string][]hdns.Record
				zones   provider.ZoneIDName
				err     bool
			}{
				err: true,
			},
		},
		{
			name: "Records error returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: testZones,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: len(testZones),
								},
							},
						},
					},
					getRecords: recordsResponse{
						err: errors.New("test records error"),
					},
					adjustZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
			},
			expected: struct {
				records map[string][]hdns.Record
				zones   provider.ZoneIDName
				err     bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_makeEndpointName tests makeEndpointName().
func Test_makeEndpointName(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			domain    string
			entryName string
			epType    string
		}
		expected string
	}

	testCases := []testCase{
		{
			name: "Adjustment not required",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "a.com",
				entryName: "test",
				epType:    "A",
			},
			expected: "test",
		},
		{
			name: "Adjustment required",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "a.com",
				entryName: "test.a.com",
				epType:    "A",
			},
			expected: "test",
		},
		{
			name: "top entry",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "a.com",
				entryName: "a.com",
				epType:    "A",
			},
			expected: "@",
		},
	}

	run := func(t *testing.T, tc testCase) {
		actual := makeEndpointName(tc.input.domain, tc.input.entryName, tc.input.epType)
		assert.Equal(t, actual, tc.expected)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_makeEndpointTarget tests HetznerProvider.makeEndpointTarget().
func Test_makeEndpointTarget(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		input    [3]string
		expected struct {
			target string
			valid  bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		target, valid := tc.provider.makeEndpointTarget(tc.input[0], tc.input[1], tc.input[2])
		assert.Equal(t, target, tc.expected.target)
		assert.Equal(t, valid, tc.expected.valid)
	}

	testCases := []testCase{
		{
			name: "No domain provided",
			provider: HetznerProvider{
				client:       &mockClient{},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
				zoneIDNameMapper: provider.ZoneIDName{
					"1": "a.com",
				},
			},
			input: [3]string{"", "www.a.com", "A"},
			expected: struct {
				target string
				valid  bool
			}{
				target: "www.a.com",
				valid:  true,
			},
		},
		{
			name: "Domain removed",
			provider: HetznerProvider{
				client:       &mockClient{},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
				zoneIDNameMapper: provider.ZoneIDName{
					"1": "a.com",
				},
			},
			input: [3]string{"a.com", "www.a.com", "A"},
			expected: struct {
				target string
				valid  bool
			}{
				target: "www",
				valid:  true,
			},
		},
		{
			name: "Trailing dot removed",
			provider: HetznerProvider{
				client:       &mockClient{},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.DomainFilter{},
				zoneIDNameMapper: provider.ZoneIDName{
					"1": "a.com",
				},
			},
			input: [3]string{"a.com", "www.", "A"},
			expected: struct {
				target string
				valid  bool
			}{
				target: "www",
				valid:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_submitChanges tests HetznerProvider.submitChanges().
func Test_submitChanges(t *testing.T) {
}

// Test_endpointsByZoneID tests endpointsByZoneID().
func Test_endpointsByZoneID(t *testing.T) {
}

// Test_getMatchingDomainRecords tests getMatchingDomainRecords().
func Test_getMatchingDomainRecords(t *testing.T) {
}

// Test_getTTLFromEndpoint tests getTTLFromEndpoint().
func Test_getTTLFromEndpoint(t *testing.T) {
}

// Test_processCreateActions tests processCreateActions().
func Test_processCreateActions(t *testing.T) {
}

// Test_processUpdateActions tests processUpdateActions().
func Test_processUpdateActions(t *testing.T) {
}

// Test_processDeleteActions tests processDeleteActions().
func Test_processDeleteActions(t *testing.T) {
}

// Test_ApplyChanges tests HetznerProvider.ApplyChanges().
func Test_ApplyChanges(t *testing.T) {
}
