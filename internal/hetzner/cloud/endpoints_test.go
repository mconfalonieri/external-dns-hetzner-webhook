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
package hetznercloud

import (
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
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

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := makeEndpointName(inp.domain, inp.entryName)
		assert.Equal(t, actual, tc.expected)
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_makeEndpointTarget tests makeEndpointTarget().
func Test_makeEndpointTarget(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected struct {
			adjustedTarget string
			err            error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual, err := makeEndpointTarget(inp)
		assert.Equal(t, exp.adjustedTarget, actual)
		assertError(t, exp.err, err)
	}

	testCases := []testCase{
		{
			name:  "ipv4 address",
			input: "1.1.1.1",
			expected: struct {
				adjustedTarget string
				err            error
			}{
				adjustedTarget: "1.1.1.1",
			},
		},
		{
			name:  "fqdn provided",
			input: "www.alpha.com",
			expected: struct {
				adjustedTarget string
				err            error
			}{
				adjustedTarget: "www.alpha.com",
			},
		},
		{
			name:  "hostname with trailing dot",
			input: "www.",
			expected: struct {
				adjustedTarget string
				err            error
			}{
				adjustedTarget: "www",
			},
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
		input    *hcloud.ZoneRRSet
		expected *endpoint.Endpoint
	}

	run := func(t *testing.T, tc testCase) {
		actual := createEndpointFromRecord("--slash--", tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "top domain",
			input: &hcloud.ZoneRRSet{
				Zone: &hcloud.Zone{
					ID:   2,
					Name: "beta.com",
				},
				ID:   "id_0",
				Name: "@",
				Type: hcloud.ZoneRRSetTypeCNAME,
				TTL:  &defaultTTL,
				Records: []hcloud.ZoneRRSetRecord{
					{
						Value: "www.alpha.com.",
					},
				},
			},
			expected: &endpoint.Endpoint{
				DNSName:    "beta.com",
				RecordType: endpoint.RecordTypeCNAME,
				Targets:    endpoint.Targets{"www.alpha.com"},
				RecordTTL:  endpoint.TTL(defaultTTL),
				Labels:     endpoint.Labels{},
			},
		},
		{
			name: "single record",
			input: &hcloud.ZoneRRSet{
				Zone: &hcloud.Zone{
					ID:   2,
					Name: "beta.com",
				},
				ID:   "id_1",
				Name: "ftp",
				Type: hcloud.ZoneRRSetTypeCNAME,
				TTL:  &defaultTTL,
				Records: []hcloud.ZoneRRSetRecord{
					{
						Value: "www.alpha.com.",
					},
				},
			},
			expected: &endpoint.Endpoint{
				DNSName:    "ftp.beta.com",
				RecordType: "CNAME",
				Targets:    endpoint.Targets{"www.alpha.com"},
				RecordTTL:  endpoint.TTL(defaultTTL),
				Labels:     endpoint.Labels{},
			},
		},
		{
			name: "multiple records",
			input: &hcloud.ZoneRRSet{
				Zone: &hcloud.Zone{
					ID:   2,
					Name: "beta.com",
				},
				ID:   "id_1",
				Name: "ftp",
				Type: hcloud.ZoneRRSetTypeCNAME,
				TTL:  &defaultTTL,
				Records: []hcloud.ZoneRRSetRecord{
					{
						Value: "www.alpha.com.",
					}, {
						Value: "www.gamma.com.",
					},
				},
			},
			expected: &endpoint.Endpoint{
				DNSName:    "ftp.beta.com",
				RecordType: "CNAME",
				Targets:    endpoint.Targets{"www.alpha.com", "www.gamma.com"},
				RecordTTL:  endpoint.TTL(defaultTTL),
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
			zoneIDNameMapper zoneIDName
			endpoints        []*endpoint.Endpoint
		}
		expected map[int64][]*endpoint.Endpoint
	}

	run := func(t *testing.T, tc testCase) {
		actual := endpointsByZoneID(tc.input.zoneIDNameMapper, tc.input.endpoints)
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name: "empty input",
			input: struct {
				zoneIDNameMapper zoneIDName
				endpoints        []*endpoint.Endpoint
			}{
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
				endpoints: []*endpoint.Endpoint{},
			},
			expected: map[int64][]*endpoint.Endpoint{},
		},
		{
			name: "some input",
			input: struct {
				zoneIDNameMapper zoneIDName
				endpoints        []*endpoint.Endpoint
			}{
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
			expected: map[int64][]*endpoint.Endpoint{
				1: {
					&endpoint.Endpoint{
						DNSName:    "www.alpha.com",
						RecordType: "A",
						Targets: endpoint.Targets{
							"127.0.0.1",
						},
					},
				},
				2: {
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

// Test_getMatchingDomainRRSet tests getMatchingDomainRRSet().
func Test_getMatchingDomainRRSet(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			rrsets   []*hcloud.ZoneRRSet
			zoneName string
			ep       *endpoint.Endpoint
		}
		expected struct {
			rrset *hcloud.ZoneRRSet
			ok    bool
		}
	}

	testCases := []testCase{
		{
			name: "no matches",
			input: struct {
				rrsets   []*hcloud.ZoneRRSet
				zoneName string
				ep       *endpoint.Endpoint
			}{
				rrsets: []*hcloud.ZoneRRSet{
					{
						Zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						ID:   "id1",
						Name: "www",
					},
				},
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName: "ftp.alpha.com",
				},
			},
			expected: struct {
				rrset *hcloud.ZoneRRSet
				ok    bool
			}{
				rrset: nil,
				ok:    false,
			},
		},
		{
			name: "matches",
			input: struct {
				rrsets   []*hcloud.ZoneRRSet
				zoneName string
				ep       *endpoint.Endpoint
			}{
				rrsets: []*hcloud.ZoneRRSet{
					{
						Zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						ID:   "id1",
						Name: "www",
					},
				},
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName: "www.alpha.com",
				},
			},
			expected: struct {
				rrset *hcloud.ZoneRRSet
				ok    bool
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id1",
					Name: "www",
				},
				ok: true,
			},
		},
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		rrset, ok := getMatchingDomainRRSet(tc.input.rrsets, tc.input.zoneName, tc.input.ep)
		assert.EqualValues(t, exp.rrset, rrset)
		assert.Equal(t, exp.ok, ok)
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

func Test_getEndpointLogFields(t *testing.T) {
	type testCase struct {
		name     string
		input    *endpoint.Endpoint
		expected log.Fields
	}

	run := func(t *testing.T, tc testCase) {
		actual := getEndpointLogFields(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "single target endpoint",
			input: &endpoint.Endpoint{
				DNSName:    "www.alpha.com",
				RecordType: "A",
				Targets:    endpoint.Targets{"1.1.1.1"},
				RecordTTL:  7200,
			},
			expected: log.Fields{
				"DNSName":    "www.alpha.com",
				"RecordType": "A",
				"Targets":    "1.1.1.1",
				"TTL":        7200,
			},
		},
		{
			name: "multiple target endpoint",
			input: &endpoint.Endpoint{
				DNSName:    "www.alpha.com",
				RecordType: "A",
				Targets:    endpoint.Targets{"1.1.1.1", "2.2.2.2"},
				RecordTTL:  7200,
			},
			expected: log.Fields{
				"DNSName":    "www.alpha.com",
				"RecordType": "A",
				"Targets":    "1.1.1.1;2.2.2.2",
				"TTL":        7200,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
