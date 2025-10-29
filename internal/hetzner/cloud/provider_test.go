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
package hetznercloud

import (
	"context"
	"errors"
	"external-dns-hetzner-webhook/internal/hetzner"
	"net/http"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/external-dns/endpoint"
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
			zones []*hcloud.Zone
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
					1: "alpha.com",
					2: "beta.com",
				},
			},
			input:    []*endpoint.Endpoint{},
			expected: []*endpoint.Endpoint{},
		},
		{
			name: "adjusted elements",
			provider: HetznerProvider{
				zoneIDNameMapper: zoneIDName{
					1: "alpha.com",
					2: "beta.com",
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
								Type: hcloud.ZoneRRSetTypeA,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "2.2.2.2",
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
		expected map[int64]string
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
			expected: map[int64]string{},
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
			expected: map[int64]string{
				1: "alpha.com",
				2: "beta.com",
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
