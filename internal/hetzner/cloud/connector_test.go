/*
 * Connector - unit tests.
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
	"net/http"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

// Test_fetchRecords tests fetchRecords().
func Test_fetchRecords(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zone      *hcloud.Zone
			client    apiClient
			batchSize int
		}
		expected struct {
			rrsets []*hcloud.ZoneRRSet
			err    error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchRecords(context.Background(), inp.zone, inp.client, inp.batchSize)
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, exp.rrsets, actual)
		}
	}

	testCases := []testCase{
		{
			name: "RRSets fetched",
			input: struct {
				zone      *hcloud.Zone
				client    apiClient
				batchSize int
			}{
				zone: &hcloud.Zone{ID: 1},
				client: &mockClient{
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
									ID:   1,
									Name: "alpha.com",
								},
								ID:   "id_3",
								Name: "mail",
								Type: hcloud.ZoneRRSetTypeMX,
								TTL:  &defaultTTL,
								Records: []hcloud.ZoneRRSetRecord{
									{
										Value: "3.3.3.3",
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
									TotalEntries: 3,
								},
							},
						},
					},
				},
				batchSize: 100,
			},
			expected: struct {
				rrsets []*hcloud.ZoneRRSet
				err    error
			}{
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
							ID:   1,
							Name: "alpha.com",
						},
						ID:   "id_3",
						Name: "mail",
						Type: hcloud.ZoneRRSetTypeMX,
						TTL:  &defaultTTL,
						Records: []hcloud.ZoneRRSetRecord{
							{
								Value: "3.3.3.3",
							},
						},
					},
				},
			},
		},
		{
			name: "error fetching records",
			input: struct {
				zone      *hcloud.Zone
				client    apiClient
				batchSize int
			}{
				zone: &hcloud.Zone{ID: 1},
				client: &mockClient{
					getRRSets: rrSetsResponse{
						err: errors.New("records test error"),
					},
				},
				batchSize: 100,
			},
			expected: struct {
				rrsets []*hcloud.ZoneRRSet
				err    error
			}{
				err: errors.New("records test error"),
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
		name  string
		input struct {
			client    apiClient
			batchSize int
		}
		expected struct {
			zones []*hcloud.Zone
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchZones(context.Background(), inp.client, inp.batchSize)
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, actual, exp.zones)
		}
	}

	testCases := []testCase{
		{
			name: "zones fetched",
			input: struct {
				client    apiClient
				batchSize int
			}{
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
				batchSize: 100,
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
			name: "error fetching zones",
			input: struct {
				client    apiClient
				batchSize int
			}{
				client: &mockClient{
					getZones: zonesResponse{
						err: errors.New("zones test error"),
					},
				},
				batchSize: 100,
			},
			expected: struct {
				zones []*hcloud.Zone
				err   error
			}{
				err: errors.New("zones test error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
