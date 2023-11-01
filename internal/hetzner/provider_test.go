package hetzner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	hschema "github.com/jobstoit/hetzner-dns-go/dns/schema"
	"gotest.tools/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

type zonesResponse struct {
	zones []*hdns.Zone
	resp  *hdns.Response
	err   error
}

type recordsResponse struct {
	records []*hdns.Record
	resp    *hdns.Response
	err     error
}

type recordResponse struct {
	record *hdns.Record
	resp   *hdns.Response
	err    error
}

type deleteResponse struct {
	resp *hdns.Response
	err  error
}

type mockClient struct {
	getZones     zonesResponse
	getRecords   recordsResponse
	createRecord recordResponse
	updateRecord recordResponse
	deleteRecord deleteResponse
	adjustZone   bool
}

func (m *mockClient) GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error) {
	r := m.getZones
	return r.zones, r.resp, r.err
}

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

func (m *mockClient) CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.createRecord
	return r.record, r.resp, r.err
}

func (m *mockClient) UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error) {
	r := m.updateRecord
	return r.record, r.resp, r.err
}

func (m *mockClient) DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error) {
	r := m.deleteRecord
	return r.resp, r.err
}

var (
	unpointedZones = func(zones []*hdns.Zone) []hdns.Zone {
		ret := make([]hdns.Zone, len(zones))
		for i, z := range zones {
			if z == nil {
				continue
			}
			ret[i] = *z
		}
		return ret
	}
	unpointedRecords = func(records []*hdns.Record) []hdns.Record {
		ret := make([]hdns.Record, len(records))
		for i, r := range records {
			if r == nil {
				continue
			}
			ret[i] = *r
		}
		return ret
	}
	checkErr = func(t *testing.T, err error, errExp bool) {
		isErr := (err != nil)
		if (isErr && !errExp) || (!isErr && errExp) {
			t.Fail()
		}
	}
	toEndpoints = func(records []hdns.Record) []*endpoint.Endpoint {
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
	testTime = time.Date(2021, 8, 15, 14, 30, 45, 100, time.Local)
)

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
	assert.DeepEqual(t, actualJSON, expectedJSON)
}

func Test_Empty(t *testing.T) {
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
		if tc.expected.err {
			checkErr(t, err, true)
		} else {
			checkErr(t, err, false)
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
		if tc.expected.err {
			checkErr(t, err, true)
		} else {
			checkErr(t, err, false)
			for i, r := range actual {
				fmt.Printf("[%d] type{%s} id{%s} name{%s} value{%v}\n", i, r.RecordType, r.SetIdentifier, r.DNSName, r.Targets)
			}
			assert.Equal(t, len(actual), tc.expected.endpoints)
		}
	}

	testCases := []testCase{
		{
			name: "All records",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_fetchRecords(t *testing.T) {
}

func Test_fetchZones(t *testing.T) {
}

func Test_ensureZoneIDMappingPresent(t *testing.T) {
}

func Test_getRecordsByZoneID(t *testing.T) {
}

func Test_makeEndpointName(t *testing.T) {
}

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
		assert.DeepEqual(t, target, tc.expected.target)
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

func Test_submitChanges(t *testing.T) {
}

func Test_endpointsByZoneID(t *testing.T) {
}

func Test_getMatchingDomainRecords(t *testing.T) {
}

func Test_getTTLFromEndpoint(t *testing.T) {
}

func Test_processCreateActions(t *testing.T) {
}

func Test_processUpdateActions(t *testing.T) {
}

func Test_processDeleteActions(t *testing.T) {
}

func Test_ApplyChanges(t *testing.T) {
}
