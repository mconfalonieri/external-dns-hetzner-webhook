/*
 * Provider - unit tests.
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
	"external-dns-hetzner-webhook/internal/hetzner"
	"net/http"
	"testing"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

// Test_NewHetznerProvider tests NewHetznerProvider().
func Test_NewHetznerProvider(t *testing.T) {
	cfg := hetzner.Configuration{
		APIKey:       "testKey",
		DryRun:       true,
		Debug:        true,
		BatchSize:    50,
		DefaultTTL:   3600,
		DomainFilter: []string{"alpha.com, beta.com"},
	}

	p, _ := NewHetznerProvider(&cfg)

	assert.Equal(t, cfg.DryRun, p.dryRun)
	assert.Equal(t, cfg.Debug, p.debug)
	assert.Equal(t, cfg.BatchSize, p.batchSize)
	assert.Equal(t, cfg.DefaultTTL, p.defaultTTL)
	actualJSON, _ := p.domainFilter.MarshalJSON()
	expectedJSON, _ := hetzner.GetDomainFilter(cfg).MarshalJSON()
	assert.Equal(t, actualJSON, expectedJSON)
}

// Test_Zones tests HetznerProvider.Zones().
func Test_Zones(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			zones []hdns.Zone
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.provider
		exp := tc.expected
		actual, err := obj.Zones(context.Background())
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, exp.zones, actual)
		}
	}

	testCases := []testCase{
		{
			name: "all zones returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
						},
						resp: &hdns.Response{
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   error
			}{
				zones: []hdns.Zone{
					{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					{
						ID:   "zoneIDBeta",
						Name: "beta.com",
					},
				},
			},
		},
		{
			name: "filtered zones returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							{
								ID:   "zoneIDGamma",
								Name: "gamma.com",
							},
						},
						resp: &hdns.Response{
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: endpoint.NewDomainFilter([]string{"alpha.com", "gamma.com"}),
			},
			expected: struct {
				zones []hdns.Zone
				err   error
			}{
				zones: []hdns.Zone{
					{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					{
						ID:   "zoneIDGamma",
						Name: "gamma.com",
					},
				},
			},
		},
		{
			name: "error returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test zones error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				zones []hdns.Zone
				err   error
			}{
				err: errors.New("test zones error"),
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

	run := func(t *testing.T, tc testCase) {
		obj := tc.provider
		inp := tc.input
		exp := tc.expected
		actual, err := obj.AdjustEndpoints(inp)
		assert.Nil(t, err) // This implementation shouldn't throw errors
		assert.EqualValues(t, exp, actual)
	}

	testCases := []testCase{
		{
			name: "empty list",
			provider: HetznerProvider{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
			},
			input:    []*endpoint.Endpoint{},
			expected: []*endpoint.Endpoint{},
		},
		{
			name: "adjusted elements",
			provider: HetznerProvider{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
			},
			input: []*endpoint.Endpoint{
				{
					DNSName:    "www.alpha.com",
					RecordType: "A",
					Targets:    endpoint.Targets{"1.1.1.1"},
				},
				{
					DNSName:    "alpha.com",
					RecordType: "CNAME",
					Targets:    endpoint.Targets{"www.alpha.com."},
				},
				{
					DNSName:    "www.beta.com",
					RecordType: "A",
					Targets:    endpoint.Targets{"2.2.2.2"},
				},
				{
					DNSName:    "ftp.beta.com",
					RecordType: "CNAME",
					Targets:    endpoint.Targets{"www.alpha.com."},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "www.alpha.com",
					RecordType: "A",
					Targets:    endpoint.Targets{"1.1.1.1"},
				},
				{
					DNSName:    "alpha.com",
					RecordType: "CNAME",
					Targets:    endpoint.Targets{"www.alpha.com"},
				},
				{
					DNSName:    "www.beta.com",
					RecordType: "A",
					Targets:    endpoint.Targets{"2.2.2.2"},
				},
				{
					DNSName:    "ftp.beta.com",
					RecordType: "CNAME",
					Targets:    endpoint.Targets{"www.alpha.com"},
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

// Test_Records tests HetznerProvider.Records().
func Test_Records(t *testing.T) {
	type testCase struct {
		name     string
		provider HetznerProvider
		expected struct {
			endpoints []*endpoint.Endpoint
			err       error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.provider
		exp := tc.expected
		actual, err := obj.Records(context.Background())
		if !assertError(t, exp.err, err) {
			assert.EqualValues(t, exp.endpoints, actual)
		}
	}

	testCases := []testCase{
		{
			name: "empty list",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: []*hdns.Record{},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0,
								},
							},
						},
					},
					filterRecordsByZone: true, // we want the records by zone
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints []*endpoint.Endpoint
				err       error
			}{
				endpoints: []*endpoint.Endpoint{},
			},
		},
		{
			name: "records returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: []*hdns.Record{
							{
								ID:   "id_1",
								Name: "www",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "1.1.1.1",
								Ttl:   -1,
							},
							{
								ID:   "id_2",
								Name: "ftp",
								Type: hdns.RecordTypeCNAME,
								Zone: &hdns.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "www",
								Ttl:   -1,
							},
							{
								ID:   "id_3",
								Name: "www",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDBeta",
									Name: "beta.com",
								},
								Value: "2.2.2.2",
								Ttl:   -1,
							},
							{
								ID:   "id_4",
								Name: "ftp",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDBeta",
									Name: "beta.com",
								},
								Value: "3.3.3.3",
								Ttl:   -1,
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0, // This value will be adjusted
								},
							},
						},
					},
					filterRecordsByZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints []*endpoint.Endpoint
				err       error
			}{
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "www.alpha.com",
						RecordType: "A",
						Targets:    endpoint.Targets{"1.1.1.1"},
						Labels:     endpoint.Labels{},
						RecordTTL:  -1,
					},
					{
						DNSName:    "ftp.alpha.com",
						RecordType: "CNAME",
						Targets:    endpoint.Targets{"www.alpha.com"},
						Labels:     endpoint.Labels{},
						RecordTTL:  -1,
					},
					{
						DNSName:    "www.beta.com",
						RecordType: "A",
						Targets:    endpoint.Targets{"2.2.2.2"},
						Labels:     endpoint.Labels{},
						RecordTTL:  -1,
					},
					{
						DNSName:    "ftp.beta.com",
						RecordType: "A",
						Targets:    endpoint.Targets{"3.3.3.3"},
						Labels:     endpoint.Labels{},
						RecordTTL:  -1,
					},
				},
			},
		},
		{
			name: "error getting zones",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test zones error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints []*endpoint.Endpoint
				err       error
			}{
				err: errors.New("test zones error"),
			},
		},
		{
			name: "error getting records",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						err: errors.New("test records error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				endpoints []*endpoint.Endpoint
				err       error
			}{
				err: errors.New("test records error"),
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
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty list",
			provider: HetznerProvider{},
			input:    []hdns.Zone{},
			expected: map[string]string{},
		},
		{
			name:     "zones present",
			provider: HetznerProvider{},
			input: []hdns.Zone{
				{
					ID:   "zoneIDAlpha",
					Name: "alpha.com",
				},
				{
					ID:   "zoneIDBeta",
					Name: "beta.com",
				},
			},
			expected: map[string]string{
				"zoneIDAlpha": "alpha.com",
				"zoneIDBeta":  "beta.com",
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
			recordsByZoneID map[string][]hdns.Record
			err             error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.provider
		exp := tc.expected
		actual, err := obj.getRecordsByZoneID(context.Background())
		if assertError(t, exp.err, err) {
			assert.ElementsMatch(t, exp.recordsByZoneID, actual)
		}
	}

	testCases := []testCase{
		{
			name: "empty list",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: []*hdns.Record{},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0,
								},
							},
						},
					},
					filterRecordsByZone: true, // we want the records by zone
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[string][]hdns.Record
				err             error
			}{
				recordsByZoneID: map[string][]hdns.Record{},
			},
		},
		{
			name: "records returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						records: []*hdns.Record{
							{
								ID:   "id_1",
								Name: "www",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "1.1.1.1",
								Ttl:   -1,
							},
							{
								ID:   "id_2",
								Name: "ftp",
								Type: hdns.RecordTypeCNAME,
								Zone: &hdns.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "www",
								Ttl:   -1,
							},
							{
								ID:   "id_3",
								Name: "www",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDBeta",
									Name: "beta.com",
								},
								Value: "2.2.2.2",
								Ttl:   -1,
							},
							{
								ID:   "id_4",
								Name: "ftp",
								Type: hdns.RecordTypeA,
								Zone: &hdns.Zone{
									ID:   "zoneIDBeta",
									Name: "beta.com",
								},
								Value: "3.3.3.3",
								Ttl:   -1,
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0, // This value will be adjusted
								},
							},
						},
					},
					filterRecordsByZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[string][]hdns.Record
				err             error
			}{
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						{
							ID:   "id_1",
							Name: "www",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
						{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeCNAME,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "www",
							Ttl:   -1,
						},
					},
					"zoneIDBeta": {
						{
							ID:   "id_3",
							Name: "www",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
						{
							ID:   "id_4",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "3.3.3.3",
							Ttl:   -1,
						},
					},
				},
			},
		},
		{
			name: "error getting zones",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("test zones error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[string][]hdns.Record
				err             error
			}{
				err: errors.New("test zones error"),
			},
		},
		{
			name: "error getting records",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hdns.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hdns.Meta{
								Pagination: &hdns.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRecords: recordsResponse{
						err: errors.New("test records error"),
					},
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[string][]hdns.Record
				err             error
			}{
				err: errors.New("test records error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
