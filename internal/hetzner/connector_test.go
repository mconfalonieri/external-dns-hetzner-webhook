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
package hetzner

import (
	"context"
	"errors"
	"net/http"
	"testing"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
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
			endpoints int
			err       bool
		}
	}

	testRecords := buildTestRecords("zoneID")

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchRecords(context.Background(), inp.zoneID, inp.dnsClient, inp.batchSize)
		checkError(t, err, exp.err)
		if err != nil {
			assert.Equal(t, len(actual), exp.endpoints)
		}
	}

	testCases := []testCase{
		{
			name: "All records",
			input: struct {
				zoneID    string
				dnsClient apiClient
				batchSize int
			}{
				zoneID: "zoneID",
				dnsClient: &mockClient{
					getRecords: recordsResponse{
						records: testRecords,
						resp: &hdns.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
						},
					},
					adjustZone: true,
				},
				batchSize: 100,
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
			input: struct {
				zoneID    string
				dnsClient apiClient
				batchSize int
			}{
				zoneID: "zoneID",
				dnsClient: &mockClient{
					getRecords: recordsResponse{
						err: errors.New("records test error"),
					},
					adjustZone: true,
				},
				batchSize: 100,
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
		name  string
		input struct {
			dnsClient apiClient
			batchSize int
		}
		expected struct {
			zones []hdns.Zone
			err   bool
		}
	}

	testZones := buildTestZones()

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := fetchZones(context.Background(), inp.dnsClient, inp.batchSize)
		checkError(t, err, exp.err)
		if err == nil {
			assert.ElementsMatch(t, actual, exp.zones)
		}
	}

	testCases := []testCase{
		{
			name: "Zones returned",
			input: struct {
				dnsClient apiClient
				batchSize int
			}{
				dnsClient: &mockClient{
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
				batchSize: 100,
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
			input: struct {
				dnsClient apiClient
				batchSize int
			}{
				dnsClient: &mockClient{
					getZones: zonesResponse{
						err: errors.New("zones test error"),
					},
					adjustZone: true,
				},
				batchSize: 100,
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
