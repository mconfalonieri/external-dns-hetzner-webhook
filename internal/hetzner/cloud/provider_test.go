/*
 * Provider - unit tests.
 *
 * Copyright 2026 Marco Confalonieri.
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
package hetznercloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/external-dns/endpoint"
)

// assertEqualDomainFilter checks that the domain filters have the same information.
func assertEqualDomainFilter(t *testing.T, expected, actual *endpoint.DomainFilter) {
	actualJSON, _ := actual.MarshalJSON()
	expectedJSON, _ := expected.MarshalJSON()
	assert.Equal(t, expectedJSON, actualJSON)
}

// assertEqualProviders checks that the fields in the actual provider match the ones
// in the expected provider, except the API client. The checked fields are:
//
//   - batchSize
//   - debug
//   - dryRun
//   - defaultTTL
//   - zoneIDNameMapper
//   - domainFilter
//   - slashEscSeq
//   - maxFailCount
//   - failCount
//   - zonesCacheDuration
//   - zonesCacheUpdate
//   - zonesCache
//
// For zonesCacheUpdate, a delta up to maxDelta is taken into consideration.
func assertEqualProviders(t *testing.T, expected, actual *HetznerProvider, maxDelta time.Duration) {
	if expected == nil && actual == nil {
		return
	} else if expected == nil && actual != nil {
		assert.Fail(t, "Expected nil provider, but it was not nil")
	} else if expected != nil && actual == nil {
		assert.Fail(t, "Expected not-nil provider, but it was nil")
	}

	if expected.client != nil {
		assert.NotNil(t, actual.client)
	} else {
		assert.Nil(t, actual.client)
	}
	assert.Equal(t, expected.batchSize, actual.batchSize)
	assert.Equal(t, expected.debug, actual.debug)
	assert.Equal(t, expected.dryRun, actual.dryRun)
	assert.Equal(t, expected.defaultTTL, actual.defaultTTL)
	assert.Equal(t, expected.zoneIDNameMapper, actual.zoneIDNameMapper)
	assertEqualDomainFilter(t, expected.domainFilter, actual.domainFilter)
	assert.Equal(t, expected.slashEscSeq, actual.slashEscSeq)
	assert.Equal(t, expected.maxFailCount, actual.maxFailCount)
	assert.Equal(t, expected.failCount, actual.failCount)
	assert.Equal(t, expected.zonesCacheDuration, actual.zonesCacheDuration)
	delta := expected.zonesCacheUpdate.Sub(actual.zonesCacheUpdate)
	assert.LessOrEqual(t, delta, maxDelta)
	assert.Equal(t, expected.zonesCache, actual.zonesCache)
}

// Test_NewHetznerProvider tests NewHetznerProvider().
func Test_NewHetznerProvider(t *testing.T) {
	type testCase struct {
		name     string
		input    *hetzner.Configuration
		expected struct {
			provider *HetznerProvider
			err      error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		provider, err := NewHetznerProvider(tc.input)
		assertError(t, exp.err, err)
		assertEqualProviders(t, exp.provider, provider, time.Second)
	}

	testCases := []testCase{
		{
			name: "empty api key",
			input: &hetzner.Configuration{
				APIKey:       "",
				DryRun:       true,
				Debug:        true,
				BatchSize:    50,
				DefaultTTL:   3600,
				DomainFilter: []string{"alpha.com, beta.com"},
				SlashEscSeq:  "--slash--",
			},
			expected: struct {
				provider *HetznerProvider
				err      error
			}{
				err: errors.New("cannot instantiate cloud DNS provider: nil API key provided"),
			},
		},
		{
			name: "some api key",
			input: &hetzner.Configuration{
				APIKey:        "TEST_API_KEY",
				DryRun:        true,
				Debug:         true,
				BatchSize:     50,
				DefaultTTL:    3600,
				DomainFilter:  []string{"alpha.com, beta.com"},
				SlashEscSeq:   "--slash--",
				MaxFailCount:  10,
				ZonesCacheTTL: 3600,
			},
			expected: struct {
				provider *HetznerProvider
				err      error
			}{
				provider: &HetznerProvider{
					client:             &mockClient{},
					batchSize:          50,
					debug:              true,
					dryRun:             true,
					defaultTTL:         3600,
					domainFilter:       endpoint.NewDomainFilter([]string{"alpha.com, beta.com"}),
					slashEscSeq:        "--slash--",
					maxFailCount:       10,
					zonesCacheDuration: time.Duration(int64(3600) * int64(time.Second)),
					zonesCacheUpdate:   time.Now(),
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
		object   HetznerProvider
		expected struct {
			zones []*hcloud.Zone
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		exp := tc.expected
		actual, err := obj.Zones(context.Background())
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, exp.zones, actual)
		}
	}

	testCases := []testCase{
		{
			name: "all zones returned",
			object: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
							{
								ID:   2,
								Name: "beta.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
				},
				batchSize:          100,
				debug:              true,
				dryRun:             false,
				defaultTTL:         7200,
				domainFilter:       &endpoint.DomainFilter{},
				zonesCacheDuration: time.Duration(0),
				zonesCacheUpdate:   time.Now(),
			},
			expected: struct {
				zones []*hcloud.Zone
				err   error
			}{
				zones: []*hcloud.Zone{
					{
						ID:   1,
						Name: "alpha.com",
					},
					{
						ID:   2,
						Name: "beta.com",
					},
				},
			},
		},
		{
			name: "filtered zones returned",
			object: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
							{
								ID:   2,
								Name: "beta.com",
							},
							{
								ID:   3,
								Name: "gamma.com",
							},
						},
						resp: &hcloud.Response{
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
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
				zones []*hcloud.Zone
				err   error
			}{
				zones: []*hcloud.Zone{
					{
						ID:   1,
						Name: "alpha.com",
					},
					{
						ID:   3,
						Name: "gamma.com",
					},
				},
			},
		},
		{
			name: "cached zones returned",
			object: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
							{
								ID:   2,
								Name: "beta.com",
							},
							{
								ID:   3,
								Name: "gamma.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 3,
								},
							},
						},
					},
				},
				batchSize:          100,
				debug:              true,
				dryRun:             false,
				defaultTTL:         7200,
				domainFilter:       &endpoint.DomainFilter{},
				zonesCacheDuration: time.Duration(int64(3600) * int64(time.Second)),
				zonesCacheUpdate:   time.Now().Add(time.Duration(int64(3600) * int64(time.Second))),
				zonesCache: []*hcloud.Zone{
					{
						ID:   1,
						Name: "alpha.com",
					},
					{
						ID:   2,
						Name: "beta.com",
					},
				},
			},
			expected: struct {
				zones []*hcloud.Zone
				err   error
			}{
				zones: []*hcloud.Zone{
					{
						ID:   1,
						Name: "alpha.com",
					},
					{
						ID:   2,
						Name: "beta.com",
					},
				},
			},
		},
		{
			name: "error returned",
			object: HetznerProvider{
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
				zones []*hcloud.Zone
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
				zoneIDNameMapper: zoneIDName{
					1: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					2: &hcloud.Zone{
						ID:   2,
						Name: "beta.com",
					},
				},
			},
			input:    []*endpoint.Endpoint{},
			expected: []*endpoint.Endpoint{},
		},
		{
			name: "adjusted elements",
			provider: HetznerProvider{
				zoneIDNameMapper: zoneIDName{
					1: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					2: &hcloud.Zone{
						ID:   2,
						Name: "beta.com",
					},
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
			assert.ElementsMatch(t, exp.endpoints, actual)
		}
	}

	testCases := []testCase{
		{
			name: "empty list",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRRSets: rrSetsResponse{
						rrsets: []*hcloud.ZoneRRSet{},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0,
								},
							},
						},
					},
					filterRRSetsByZone: true, // we want the records by zone
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
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
							{
								ID:   2,
								Name: "beta.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRRSets: rrSetsResponse{
						rrsets: []*hcloud.ZoneRRSet{
							{
								Zone: &hcloud.Zone{
									ID:   1,
									Name: "alpha.com",
								},
								ID:   "id_1",
								Name: "www",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "1.1.1.1",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   1,
									Name: "alpha.com",
								},
								ID:   "id_2",
								Name: "ftp",
								Type: hcloud.ZoneRRSetTypeCNAME,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "www",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   2,
									Name: "beta.com",
								},
								ID:   "id_3",
								Name: "www",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "3.3.3.3",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   2,
									Name: "beta.com",
								},
								ID:   "id_4",
								Name: "ftp",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "4.4.4.4",
									},
								},
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0, // This value will be adjusted
								},
							},
						},
					},
					filterRRSetsByZone: true,
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
						RecordType: endpoint.RecordTypeA,
						Targets:    endpoint.Targets{"1.1.1.1"},
						Labels:     endpoint.Labels{},
						RecordTTL:  endpoint.TTL(defaultTTL),
					},
					{
						DNSName:    "ftp.alpha.com",
						RecordType: endpoint.RecordTypeCNAME,
						Targets:    endpoint.Targets{"www.alpha.com"},
						Labels:     endpoint.Labels{},
						RecordTTL:  endpoint.TTL(defaultTTL),
					},
					{
						DNSName:    "www.beta.com",
						RecordType: endpoint.RecordTypeA,
						Targets:    endpoint.Targets{"3.3.3.3"},
						Labels:     endpoint.Labels{},
						RecordTTL:  endpoint.TTL(defaultTTL),
					},
					{
						DNSName:    "ftp.beta.com",
						RecordType: endpoint.RecordTypeA,
						Targets:    endpoint.Targets{"4.4.4.4"},
						Labels:     endpoint.Labels{},
						RecordTTL:  endpoint.TTL(defaultTTL),
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
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRRSets: rrSetsResponse{
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
		input    []*hcloud.Zone
		expected zoneIDName
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
			input:    []*hcloud.Zone{},
			expected: zoneIDName(map[int64]*hcloud.Zone{}),
		},
		{
			name:     "zones present",
			provider: HetznerProvider{},
			input: []*hcloud.Zone{
				{
					ID:   1,
					Name: "alpha.com",
				},
				{
					ID:   2,
					Name: "beta.com",
				},
			},
			expected: zoneIDName(map[int64]*hcloud.Zone{
				1: {
					ID:   1,
					Name: "alpha.com",
				},
				2: {
					ID:   2,
					Name: "beta.com",
				},
			}),
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
			recordsByZoneID map[int64][]*hcloud.ZoneRRSet
			err             error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.provider
		exp := tc.expected
		actual, err := obj.getRRSetsByZoneID(context.Background())
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
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRRSets: rrSetsResponse{
						rrsets: []*hcloud.ZoneRRSet{},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0,
								},
							},
						},
					},
					filterRRSetsByZone: true, // we want the records by zone
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[int64][]*hcloud.ZoneRRSet
				err             error
			}{
				recordsByZoneID: map[int64][]*hcloud.ZoneRRSet{},
			},
		},
		{
			name: "records returned",
			provider: HetznerProvider{
				client: &mockClient{
					getZones: zonesResponse{
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
							{
								ID:   2,
								Name: "beta.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 2,
								},
							},
						},
					},
					getRRSets: rrSetsResponse{
						rrsets: []*hcloud.ZoneRRSet{
							{
								Zone: &hcloud.Zone{
									ID:   1,
									Name: "alpha.com",
								},
								ID:   "id_1",
								Name: "www",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "1.1.1.1",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   1,
									Name: "alpha.com",
								},
								ID:   "id_2",
								Name: "ftp",
								Type: hcloud.ZoneRRSetTypeCNAME,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "www",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   2,
									Name: "beta.com",
								},
								ID:   "id_3",
								Name: "www",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "3.3.3.3",
									},
								},
							},
							{
								Zone: &hcloud.Zone{
									ID:   2,
									Name: "beta.com",
								},
								ID:   "id_4",
								Name: "ftp",
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "4.4.4.4",
									},
								},
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							Meta: hcloud.Meta{
								Pagination: &hcloud.Pagination{
									Page:         1,
									PerPage:      100,
									LastPage:     1,
									TotalEntries: 0, // This value will be adjusted
								},
							},
						},
					},
					filterRRSetsByZone: true,
				},
				batchSize:    100,
				debug:        true,
				dryRun:       false,
				defaultTTL:   7200,
				domainFilter: &endpoint.DomainFilter{},
			},
			expected: struct {
				recordsByZoneID map[int64][]*hcloud.ZoneRRSet
				err             error
			}{
				recordsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_2",
							Name: "ftp",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "www",
								},
							},
						},
					},
					2: {
						{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "3.3.3.3",
								},
							},
						},
						{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
							},
							ID:   "id_4",
							Name: "ftp",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "4.4.4.4",
								},
							},
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
				recordsByZoneID map[int64][]*hcloud.ZoneRRSet
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
						zones: []*hcloud.Zone{
							{
								ID:   1,
								Name: "alpha.com",
							},
						},
						resp: &hcloud.Response{
							Response: &http.Response{StatusCode: http.StatusInternalServerError},
						},
					},
					getRRSets: rrSetsResponse{
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
				recordsByZoneID map[int64][]*hcloud.ZoneRRSet
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

// Test_incFailCount tests incFailCount().
func Test_incFailCount(t *testing.T) {
	type testCase struct {
		name     string
		object   *HetznerProvider
		expected struct {
			failCount       int
			logFatalfCalled bool
			logFatalfMsg    string
		}
	}

	run := func(t *testing.T, tc testCase) {
		origLogFatalf := logFatalf
		obj := tc.object
		exp := tc.expected
		logFatalfCalled := false
		logFatalfMsg := ""

		// Mock logFatalf
		logFatalf = func(format string, a ...interface{}) {
			logFatalfCalled = true
			logFatalfMsg = fmt.Sprintf(format, a...)
		}
		// Do the call
		obj.incFailCount()
		// Restore logFatalf
		logFatalf = origLogFatalf

		assert.Equal(t, exp.failCount, obj.failCount)
		assert.Equal(t, exp.logFatalfCalled, logFatalfCalled)
		assert.Equal(t, exp.logFatalfMsg, logFatalfMsg)
	}

	testCases := []testCase{
		{
			name: "failCount is disabled",
			object: &HetznerProvider{
				maxFailCount: -1,
				failCount:    -1, // impossible value, but will not be reset if disabled
			},
			expected: struct {
				failCount       int
				logFatalfCalled bool
				logFatalfMsg    string
			}{
				failCount: -1,
			},
		},
		{
			name: "failCount is enabled and zero",
			object: &HetznerProvider{
				maxFailCount: 3,
				failCount:    0,
			},
			expected: struct {
				failCount       int
				logFatalfCalled bool
				logFatalfMsg    string
			}{
				failCount: 1,
			},
		},
		{
			name: "failCount is enabled and low",
			object: &HetznerProvider{
				maxFailCount: 3,
				failCount:    1,
			},
			expected: struct {
				failCount       int
				logFatalfCalled bool
				logFatalfMsg    string
			}{
				failCount: 2,
			},
		},
		{
			name: "failCount is enabled and high",
			object: &HetznerProvider{
				maxFailCount: 3,
				failCount:    2,
			},
			expected: struct {
				failCount       int
				logFatalfCalled bool
				logFatalfMsg    string
			}{
				failCount:       3,
				logFatalfCalled: true,
				logFatalfMsg:    "Failure count reached 3. Shutting down container.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_resetFailCount tests resetFailCount().
func Test_resetFailCount(t *testing.T) {
	type testCase struct {
		name     string
		object   *HetznerProvider
		expected int
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		exp := tc.expected
		obj.resetFailCount()
		assert.Equal(t, exp, obj.failCount)
	}

	testCases := []testCase{
		{
			name: "failCount is disabled",
			object: &HetznerProvider{
				maxFailCount: -1,
				failCount:    -1, // impossible value, but will not be reset if disabled
			},
			expected: -1,
		},
		{
			name: "failCount is enabled and zero",
			object: &HetznerProvider{
				maxFailCount: 3,
				failCount:    0,
			},
			expected: 0,
		},
		{
			name: "failCount is enabled and not zero",
			object: &HetznerProvider{
				maxFailCount: 3,
				failCount:    2,
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
