/*
 * Endpoints - unit tests.
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
	"testing"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

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
			name: "no adjustment required",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "alpha.com",
				entryName: "test",
				epType:    "A",
			},
			expected: "test",
		},
		{
			name: "stripping domain from name",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "alpha.com",
				entryName: "test.alpha.com",
				epType:    "A",
			},
			expected: "test",
		},
		{
			name: "top entry adjustment",
			input: struct {
				domain    string
				entryName string
				epType    string
			}{
				domain:    "alpha.com",
				entryName: "alpha.com",
				epType:    "A",
			},
			expected: "@",
		},
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := makeEndpointName(inp.domain, inp.entryName)
		assert.Equal(t, actual, tc.expected)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_makeEndpointTarget tests makeEndpointTarget().
func Test_makeEndpointTarget(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			domain      string
			entryTarget string
			epType      string
		}
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual := makeEndpointTarget(inp.domain, inp.entryTarget, inp.epType)
		assert.Equal(t, exp, actual)
	}

	testCases := []testCase{
		{
			name: "IP without domain provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "",
				entryTarget: "0.0.0.0",
				epType:      "A",
			},
			expected: "0.0.0.0",
		},
		{
			name: "IP with domain provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "alpha.com",
				entryTarget: "0.0.0.0",
				epType:      "A",
			},
			expected: "0.0.0.0",
		},
		{
			name: "No domain provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "",
				entryTarget: "www.alpha.com",
				epType:      "CNAME",
			},
			expected: "www.alpha.com",
		},
		{
			name: "Domain provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "alpha.com",
				entryTarget: "www.alpha.com",
				epType:      "CNAME",
			},
			expected: "www",
		},
		{
			name: "Other domain without trailing dot provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "alpha.com",
				entryTarget: "www.beta.com",
				epType:      "CNAME",
			},
			expected: "www.beta.com",
		},
		{
			name: "Other domain with trailing dot provided",
			input: struct {
				domain      string
				entryTarget string
				epType      string
			}{
				domain:      "alpha.com",
				entryTarget: "www.beta.com.",
				epType:      "CNAME",
			},
			expected: "www.beta.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_mergeEndpointsByNameType tests mergeEndpointsByNameType().
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

func Test_createEndpointFromRecord(t *testing.T) {
	type testCase struct {
		name     string
		input    hdns.Record
		expected *endpoint.Endpoint
	}

	run := func(t *testing.T, tc testCase) {
		actual := createEndpointFromRecord(tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "top domain",
			input: hdns.Record{
				ID:    "id_0",
				Name:  "@",
				Type:  hdns.RecordTypeCNAME,
				Value: "www.alpha.com",
				Zone: &hdns.Zone{
					ID:   "zoneIDBeta",
					Name: "beta.com",
				},
				Ttl: 7200,
			},
			expected: &endpoint.Endpoint{
				DNSName:    "beta.com",
				RecordType: "CNAME",
				Targets:    endpoint.Targets{"www.alpha.com"},
				RecordTTL:  7200,
				Labels:     endpoint.Labels{},
			},
		},
		{
			name: "record",
			input: hdns.Record{
				ID:    "id_1",
				Name:  "ftp",
				Type:  hdns.RecordTypeCNAME,
				Value: "www.alpha.com",
				Zone: &hdns.Zone{
					ID:   "zoneIDBeta",
					Name: "beta.com",
				},
				Ttl: 7200,
			},
			expected: &endpoint.Endpoint{
				DNSName:    "ftp.beta.com",
				RecordType: "CNAME",
				Targets:    endpoint.Targets{"www.alpha.com"},
				RecordTTL:  7200,
				Labels:     endpoint.Labels{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_endpointsByZoneID tests endpointsByZoneID().
func Test_endpointsByZoneID(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneIDNameMapper provider.ZoneIDName
			endpoints        []*endpoint.Endpoint
		}
		expected map[string][]*endpoint.Endpoint
	}

	run := func(t *testing.T, tc testCase) {
		actual := endpointsByZoneID(tc.input.zoneIDNameMapper, tc.input.endpoints)
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name: "empty input",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				endpoints        []*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				endpoints: []*endpoint.Endpoint{},
			},
			expected: map[string][]*endpoint.Endpoint{},
		},
		{
			name: "some input",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				endpoints        []*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "www.alpha.com",
						RecordType: "A",
						Targets: endpoint.Targets{
							"127.0.0.1",
						},
					},
					{
						DNSName:    "www.beta.com",
						RecordType: "A",
						Targets: endpoint.Targets{
							"127.0.0.1",
						},
					},
				},
			},
			expected: map[string][]*endpoint.Endpoint{
				"zoneIDAlpha": {
					&endpoint.Endpoint{
						DNSName:    "www.alpha.com",
						RecordType: "A",
						Targets: endpoint.Targets{
							"127.0.0.1",
						},
					},
				},
				"zoneIDBeta": {
					&endpoint.Endpoint{
						DNSName:    "www.beta.com",
						RecordType: "A",
						Targets: endpoint.Targets{
							"127.0.0.1",
						},
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

// Test_getMatchingDomainRecords tests getMatchingDomainRecords().
func Test_getMatchingDomainRecords(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			records  []hdns.Record
			zoneName string
			ep       *endpoint.Endpoint
		}
		expected []hdns.Record
	}

	testCases := []testCase{
		{
			name: "no matches",
			input: struct {
				records  []hdns.Record
				zoneName string
				ep       *endpoint.Endpoint
			}{
				records: []hdns.Record{
					{
						ID: "id1",
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Name: "www",
					},
				},
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName: "ftp.alpha.com",
				},
			},
			expected: []hdns.Record{},
		},
		{
			name: "matches",
			input: struct {
				records  []hdns.Record
				zoneName string
				ep       *endpoint.Endpoint
			}{
				records: []hdns.Record{
					{
						ID: "id1",
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Name: "www",
					},
				},
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName: "www.alpha.com",
				},
			},
			expected: []hdns.Record{
				{
					ID: "id1",
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					Name: "www",
				},
			},
		},
	}

	run := func(t *testing.T, tc testCase) {
		actual := getMatchingDomainRecords(tc.input.records, tc.input.zoneName, tc.input.ep)
		assert.ElementsMatch(t, actual, tc.expected)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getEndpointTTL tests getEndpointTTL().
func Test_getEndpointTTL(t *testing.T) {
	type testCase struct {
		name     string
		input    *endpoint.Endpoint
		expected *int
	}
	configuredTTL := 7200

	run := func(t *testing.T, tc testCase) {
		actualTTL := getEndpointTTL(tc.input)
		assert.EqualValues(t, tc.expected, actualTTL)
	}

	testCases := []testCase{
		{
			name: "TTL configured",
			input: &endpoint.Endpoint{
				RecordTTL: 7200,
			},
			expected: &configuredTTL,
		},
		{
			name: "TTL not configured",
			input: &endpoint.Endpoint{
				RecordTTL: -1,
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
