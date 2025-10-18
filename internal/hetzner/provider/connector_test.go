/*
 * Connector - unit tests.
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
package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"external-dns-hetzner-webhook/internal/hetzner/model"

	"github.com/stretchr/testify/assert"
)

// Test_fetchRecords tests fetchRecords().
func Test_fetchRecords(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneID    string
			dnsClient apiClient
			batchSize int
		}
		expected struct {
			records []model.Record
			err     error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchRecords(context.Background(), inp.zoneID, inp.dnsClient, inp.batchSize)
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, exp.records, actual)
		}
	}

	testCases := []testCase{
		{
			name: "records fetched",
			input: struct {
				zoneID    string
				dnsClient apiClient
				batchSize int
			}{
				zoneID: "zoneIDAlpha",
				dnsClient: &mockClient{
					getRecords: recordsResponse{
						records: []model.Record{
							{
								ID:   "id_1",
								Name: "www",
								Type: "A",
								Zone: &model.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "1.1.1.1",
								TTL:   -1,
							},
							{
								ID:   "id_2",
								Name: "ftp",
								Type: "A",
								Zone: &model.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "2.2.2.2",
								TTL:   -1,
							},
							{
								ID:   "id_3",
								Name: "mail",
								Type: "MX",
								Zone: &model.Zone{
									ID:   "zoneIDAlpha",
									Name: "alpha.com",
								},
								Value: "3.3.3.3",
								TTL:   -1,
							},
						},
						resp: &http.Response{StatusCode: http.StatusOK},
						pagination: &model.Pagination{
							PageIdx:      1,
							LastPage:     1,
							ItemsPerPage: 100,
							TotalCount:   3,
						},
					},
				},
				batchSize: 100,
			},
			expected: struct {
				records []model.Record
				err     error
			}{
				records: []model.Record{
					{
						ID:   "id_1",
						Name: "www",
						Type: "A",
						Zone: &model.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "1.1.1.1",
						TTL:   -1,
					},
					{
						ID:   "id_2",
						Name: "ftp",
						Type: "A",
						Zone: &model.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "2.2.2.2",
						TTL:   -1,
					},
					{
						ID:   "id_3",
						Name: "mail",
						Type: "MX",
						Zone: &model.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "3.3.3.3",
						TTL:   -1,
					},
				},
			},
		},
		{
			name: "error fetching records",
			input: struct {
				zoneID    string
				dnsClient apiClient
				batchSize int
			}{
				zoneID: "zoneIDAlpha",
				dnsClient: &mockClient{
					getRecords: recordsResponse{
						err: errors.New("records test error"),
					},
				},
				batchSize: 100,
			},
			expected: struct {
				records []model.Record
				err     error
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
			dnsClient apiClient
			batchSize int
		}
		expected struct {
			zones []model.Zone
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchZones(context.Background(), inp.dnsClient, inp.batchSize)
		if !assertError(t, exp.err, err) {
			assert.ElementsMatch(t, actual, exp.zones)
		}
	}

	testCases := []testCase{
		{
			name: "zones fetched",
			input: struct {
				dnsClient apiClient
				batchSize int
			}{
				dnsClient: &mockClient{
					getZones: zonesResponse{
						zones: []model.Zone{
							{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
						},
						resp: &http.Response{StatusCode: http.StatusOK},
						pagination: &model.Pagination{
							PageIdx:      1,
							ItemsPerPage: 100,
							LastPage:     1,
							TotalCount:   2,
						},
					},
				},
				batchSize: 100,
			},
			expected: struct {
				zones []model.Zone
				err   error
			}{
				zones: []model.Zone{
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
			name: "error fetching zones",
			input: struct {
				dnsClient apiClient
				batchSize int
			}{
				dnsClient: &mockClient{
					getZones: zonesResponse{
						err: errors.New("zones test error"),
					},
				},
				batchSize: 100,
			},
			expected: struct {
				zones []model.Zone
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
