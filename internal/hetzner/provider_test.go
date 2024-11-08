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

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

// Test_NewHetznerProvider tests NewHetznerProvider().
func Test_NewHetznerProvider(t *testing.T) {
	cfg := Configuration{
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
	expectedJSON, _ := GetDomainFilter(cfg).MarshalJSON()
	assert.Equal(t, actualJSON, expectedJSON)
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
	testRecords := buildTestRecords("zoneIDAlpha")

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
			records map[string][]hdns.Record
			zones   provider.ZoneIDName
			err     bool
		}
	}

	testZones := buildTestZones()
	testRecords := buildTestRecords("zoneIDAlpha")

	run := func(t *testing.T, tc testCase) {
		p := tc.provider
		actualRecords, err := p.getRecordsByZoneID(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			assert.Equal(t, len(actualRecords["zoneIDAlpha"]), len(testRecords))
			assert.Equal(t, len(actualRecords["zoneIDBeta"]), len(testRecords))
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
					"zoneIDAlpha": unpointedRecords(buildTestRecords("zoneIDAlpha")),
					"zoneIDBeta":  unpointedRecords(buildTestRecords("zoneIDBeta")),
				},
				zones: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
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
