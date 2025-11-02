/*
 * Change processors - unit tests.
 *
 * Copyright 2023 Marco Confalonieri.
 * Copyright 2017 The Kubernetes Authors.
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
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
)

// assertEqualChanges checks that two hetznerChanges objects contain the same
// elements.
func assertEqualChanges(t *testing.T, expected, actual hetznerChanges) {
	assert.Equal(t, expected.dryRun, actual.dryRun)
	assert.ElementsMatch(t, expected.creates, actual.creates)
	assert.ElementsMatch(t, expected.updates, actual.updates)
	assert.ElementsMatch(t, expected.deletes, actual.deletes)
}

// Test_adjustCNAMETarget tests adjustCNAMETarget()
func Test_adjustCNAMETarget(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			domain string
			target string
		}
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := adjustCNAMETarget(inp.domain, inp.target)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "target matches domain",
			input: struct {
				domain string
				target string
			}{
				domain: "alpha.com",
				target: "www.alpha.com",
			},
			expected: "www",
		},
		{
			name: "target matches domain with dot",
			input: struct {
				domain string
				target string
			}{
				domain: "alpha.com",
				target: "www.alpha.com.",
			},
			expected: "www",
		},
		{
			name: "target without dot does not match domain",
			input: struct {
				domain string
				target string
			}{
				domain: "alpha.com",
				target: "www.beta.com",
			},
			expected: "www.beta.com.",
		},
		{
			name: "target with dot does not match domain",
			input: struct {
				domain string
				target string
			}{
				domain: "alpha.com",
				target: "www.beta.com.",
			},
			expected: "www.beta.com.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_extractRRSetRecords tests extractRRSetRecords().
func Test_extractRRSetRecords(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneName string
			ep       *endpoint.Endpoint
		}
		expected []hcloud.ZoneRRSetRecord
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := extractRRSetRecords(inp.zoneName, inp.ep)
		assert.ElementsMatch(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "record type A",
			input: struct {
				zoneName string
				ep       *endpoint.Endpoint
			}{
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					Targets:    endpoint.Targets{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
					RecordType: endpoint.RecordTypeA,
				},
			},
			expected: []hcloud.ZoneRRSetRecord{
				{
					Value: "1.1.1.1",
				},
				{
					Value: "2.2.2.2",
				},
				{
					Value: "3.3.3.3",
				},
			},
		},
		{
			name: "record type CNAME same domain",
			input: struct {
				zoneName string
				ep       *endpoint.Endpoint
			}{
				zoneName: "alpha.com",
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					Targets:    endpoint.Targets{"ftp.alpha.com"},
					RecordType: endpoint.RecordTypeCNAME,
				},
			},
			expected: []hcloud.ZoneRRSetRecord{
				{
					Value: "ftp",
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

// Test_processCreateActionsByZone tests processCreateActionsByZone().
func Test_processCreateActionsByZone(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zone      *hcloud.Zone
			rrsets    []*hcloud.ZoneRRSet
			endpoints []*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		changes := &hetznerChanges{}
		processCreateActionsByZone(inp.zone, inp.rrsets, inp.endpoints, changes)
		assertEqualChanges(t, tc.expectedChanges, *changes)
	}

	testCases := []testCase{
		{
			name: "record already created",
			input: struct {
				zone      *hcloud.Zone
				rrsets    []*hcloud.ZoneRRSet
				endpoints []*endpoint.Endpoint
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
				rrsets: []*hcloud.ZoneRRSet{
					{
						Zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						ID:   "id_1",
						Type: "A",
						Name: "www",
						TTL:  &defaultTTL,
						Records: []hcloud.ZoneRRSetRecord{
							{
								Value: "127.0.0.1",
							},
						},
					},
				},
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "www.alpha.com",
						Targets:    endpoint.Targets{"127.0.0.1"},
						RecordType: "A",
						RecordTTL:  7200,
					},
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "new record created",
			input: struct {
				zone      *hcloud.Zone
				rrsets    []*hcloud.ZoneRRSet
				endpoints []*endpoint.Endpoint
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
				rrsets: []*hcloud.ZoneRRSet{
					{
						Zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						ID:   "id_1",
						Type: "A",
						Name: "www",
						TTL:  &defaultTTL,
						Records: []hcloud.ZoneRRSetRecord{
							{
								Value: "127.0.0.1",
							},
						},
					},
				},
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "ftp.alpha.com",
						Targets:    endpoint.Targets{"www.alpha.com"},
						RecordType: "CNAME",
						RecordTTL:  endpoint.TTL(defaultTTL),
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Type: hcloud.ZoneRRSetTypeCNAME,
							Name: "ftp",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "www",
								},
							},
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

// Test_processCreateActions tests processCreateActions().
func Test_processCreateActions(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneIDNameMapper zoneIDName
			rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
			createsByZoneID  map[int64][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		changes := hetznerChanges{}
		processCreateActions(inp.zoneIDNameMapper, inp.rrSetsByZoneID,
			inp.createsByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				createsByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "www",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
						},
					},
				},
				createsByZoneID: map[int64][]*endpoint.Endpoint{},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "empty changeset with key present",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				createsByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "www",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
						},
					},
				},
				createsByZoneID: map[int64][]*endpoint.Endpoint{
					1: {},
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "RRSet already created",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				createsByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "www",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
						},
					},
				},
				createsByZoneID: map[int64][]*endpoint.Endpoint{
					1: {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"127.0.0.1"},
							RecordType: "A",
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "new record created",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				createsByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
				createsByZoneID: map[int64][]*endpoint.Endpoint{
					1: {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"2.2.2.2"},
							RecordType: "A",
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Type: "A",
							Name: "www",
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "2.2.2.2",
								},
							},
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

// Test_sameZoneRRSetRecords tests sameZoneRRSetRecords().
func Test_sameZoneRRSetRecords(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			first  []hcloud.ZoneRRSetRecord
			second []hcloud.ZoneRRSetRecord
		}
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := sameZoneRRSetRecords(inp.first, inp.second)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "empty equality",
			input: struct {
				first  []hcloud.ZoneRRSetRecord
				second []hcloud.ZoneRRSetRecord
			}{
				first:  []hcloud.ZoneRRSetRecord{},
				second: []hcloud.ZoneRRSetRecord{},
			},
			expected: true,
		},
		{
			name: "equality",
			input: struct {
				first  []hcloud.ZoneRRSetRecord
				second []hcloud.ZoneRRSetRecord
			}{
				first: []hcloud.ZoneRRSetRecord{
					{
						Value: "1.1.1.1",
					},
					{
						Value: "2.2.2.2",
					},
					{
						Value: "3.3.3.3",
					},
				},
				second: []hcloud.ZoneRRSetRecord{
					{
						Value: "2.2.2.2",
					},
					{
						Value: "3.3.3.3",
					},
					{
						Value: "1.1.1.1",
					},
				},
			},
			expected: true,
		},
		{
			name: "dimension mismatch",
			input: struct {
				first  []hcloud.ZoneRRSetRecord
				second []hcloud.ZoneRRSetRecord
			}{
				first: []hcloud.ZoneRRSetRecord{
					{
						Value: "1.1.1.1",
					},
					{
						Value: "2.2.2.2",
					},
					{
						Value: "3.3.3.3",
					},
				},
				second: []hcloud.ZoneRRSetRecord{
					{
						Value: "2.2.2.2",
					},
					{
						Value: "3.3.3.3",
					},
				},
			},
			expected: false,
		},
		{
			name: "different elements",
			input: struct {
				first  []hcloud.ZoneRRSetRecord
				second []hcloud.ZoneRRSetRecord
			}{
				first: []hcloud.ZoneRRSetRecord{
					{
						Value: "1.1.1.1",
					},
					{
						Value: "2.2.2.2",
					},
					{
						Value: "3.3.3.3",
					},
				},
				second: []hcloud.ZoneRRSetRecord{
					{
						Value: "2.2.2.2",
					},
					{
						Value: "4.4.4.4",
					},
					{
						Value: "1.1.1.1",
					},
				},
			},
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_ensureStringMap tests ensureStringMap().
func Test_ensureStringMap(t *testing.T) {
	type testCase struct {
		name     string
		input    map[string]string
		expected map[string]string
	}

	run := func(t *testing.T, tc testCase) {
		actual := ensureStringMap(tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "map with elements",
			input: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			expected: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_equalStringMaps tests equalStringMaps().
func Test_equalStringMaps(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			first  map[string]string
			second map[string]string
		}
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := equalStringMaps(inp.first, inp.second)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "empty equality",
			input: struct {
				first  map[string]string
				second map[string]string
			}{
				first:  map[string]string{},
				second: map[string]string{},
			},
			expected: true,
		},
		{
			name: "equality",
			input: struct {
				first  map[string]string
				second map[string]string
			}{
				first: map[string]string{
					"label1": "value1",
					"label2": "value2",
					"label3": "value3",
				},
				second: map[string]string{
					"label1": "value1",
					"label3": "value3",
					"label2": "value2",
				},
			},
			expected: true,
		},
		{
			name: "dimension mismatch",
			input: struct {
				first  map[string]string
				second map[string]string
			}{
				first: map[string]string{
					"label1": "value1",
					"label2": "value2",
					"label3": "value3",
				},
				second: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
			expected: false,
		}, {
			name: "different elements",
			input: struct {
				first  map[string]string
				second map[string]string
			}{
				first: map[string]string{
					"label1": "value1",
					"label2": "value2",
					"label3": "value3",
				},
				second: map[string]string{
					"label1": "value1",
					"label2": "value2",
					"label4": "value4",
				},
			},
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_processUpdateEndpoint tests processUpdateEndpoint().
func Test_processUpdateEndpoint(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			mRRSet *hcloud.ZoneRRSet
			ep     *endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateEndpoint(inp.mRRSet, inp.ep, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "TTL changed",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testFirstTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1"},
					RecordTTL:  endpoint.TTL(testSecondTTL),
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Name: "ftp",
							Type: "A",
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
						ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
							TTL: &testSecondTTL,
						},
					},
				},
			},
		},
		{
			name: "records changed",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"2.2.2.2"},
					RecordTTL:  endpoint.TTL(testTTL),
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "2.2.2.2",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "records added",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1", "2.2.2.2"},
					RecordTTL:  endpoint.TTL(testTTL),
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "2.2.2.2",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record removed",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
						{
							Value: "2.2.2.2",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1"},
					RecordTTL:  endpoint.TTL(testTTL),
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "2.2.2.2",
								},
							},
						},
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "labels added",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
						{
							Value: "2.2.2.2",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1", "2.2.2.2"},
					RecordTTL:  endpoint.TTL(testTTL),
					ProviderSpecific: endpoint.ProviderSpecific{
						{
							Name:  "hetzner-labels",
							Value: "env=production",
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "2.2.2.2",
								},
							},
						},
						updateOpts: &hcloud.ZoneRRSetUpdateOpts{
							Labels: map[string]string{
								"env": "production",
							},
						},
					},
				},
			},
		},
		{
			name: "labels changed",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
						{
							Value: "2.2.2.2",
						},
					},
					Labels: map[string]string{
						"env": "test",
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1", "2.2.2.2"},
					RecordTTL:  endpoint.TTL(testTTL),
					ProviderSpecific: endpoint.ProviderSpecific{
						{
							Name:  "hetzner-labels",
							Value: "env=production",
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "2.2.2.2",
								},
							},
							Labels: map[string]string{
								"env": "test",
							},
						},
						updateOpts: &hcloud.ZoneRRSetUpdateOpts{
							Labels: map[string]string{
								"env": "production",
							},
						},
					},
				},
			},
		},
		{
			name: "all changed",
			input: struct {
				mRRSet *hcloud.ZoneRRSet
				ep     *endpoint.Endpoint
			}{
				mRRSet: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					ID:   "id_1",
					Type: "A",
					Name: "ftp",
					TTL:  &testFirstTTL,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
						{
							Value: "2.2.2.2",
						},
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1", "3.3.3.3"},
					RecordTTL:  endpoint.TTL(testSecondTTL),
					ProviderSpecific: endpoint.ProviderSpecific{
						{
							Name:  "hetzner-labels",
							Value: "env=production",
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_1",
							Type: "A",
							Name: "ftp",
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "2.2.2.2",
								},
							},
						},
						ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
							TTL: &testSecondTTL,
						},
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
								{
									Value: "3.3.3.3",
								},
							},
						},
						updateOpts: &hcloud.ZoneRRSetUpdateOpts{
							Labels: map[string]string{
								"env": "production",
							},
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

// Test_processUpdateActionsByZone tests processUpdateActionsByZone().
func Test_processUpdateActionsByZone(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zone      *hcloud.Zone
			rrsets    []*hcloud.ZoneRRSet
			endpoints []*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateActionsByZone(inp.zone, inp.rrsets, inp.endpoints, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zone      *hcloud.Zone
				rrsets    []*hcloud.ZoneRRSet
				endpoints []*endpoint.Endpoint
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
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
				},
				endpoints: []*endpoint.Endpoint{},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "mixed changeset",
			input: struct {
				zone      *hcloud.Zone
				rrsets    []*hcloud.ZoneRRSet
				endpoints []*endpoint.Endpoint
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
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
						TTL:  &testFirstTTL,
						Records: []hcloud.ZoneRRSetRecord{
							{
								Value: "2.2.2.2",
							},
						},
					},
				},
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "www.alpha.com",
						RecordType: endpoint.RecordTypeA,
						RecordTTL:  endpoint.TTL(defaultTTL),
						Targets:    []string{"3.3.3.3"},
					},
					{
						DNSName:    "ftp.alpha.com",
						RecordType: endpoint.RecordTypeA,
						RecordTTL:  endpoint.TTL(testSecondTTL),
						Targets:    []string{"2.2.2.2"},
					},
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
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
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "3.3.3.3",
								},
							},
						},
					},
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							ID:   "id_2",
							Name: "ftp",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "2.2.2.2",
								},
							},
						},
						ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
							TTL: &testSecondTTL,
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

// Test_processUpdateActions tests processUpdateActions().
func Test_processUpdateActions(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneIDNameMapper zoneIDName
			rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
			updatesByZoneID  map[int64][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateActions(inp.zoneIDNameMapper, inp.rrSetsByZoneID, inp.updatesByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				updatesByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						&hcloud.ZoneRRSet{
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
					},
					2: {
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
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
					},
				},
				updatesByZoneID: map[int64][]*endpoint.Endpoint{},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "empty changeset with key present",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				updatesByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						&hcloud.ZoneRRSet{
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
					},
					2: {
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
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
					},
				},
				updatesByZoneID: map[int64][]*endpoint.Endpoint{
					1: {},
					2: {},
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "mixed changeset",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				updatesByZoneID  map[int64][]*endpoint.Endpoint
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
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},

							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
					2: {
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
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
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Labels: map[string]string{
								"env": "test",
							},
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "ftp",
								},
							},
						},
					},
				},
				updatesByZoneID: map[int64][]*endpoint.Endpoint{
					1: {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    []string{"1.1.1.1"},
							RecordType: endpoint.RecordTypeA,
							RecordTTL:  endpoint.TTL(testSecondTTL),
						},
					},
					2: {
						&endpoint.Endpoint{
							DNSName:    "ftp.beta.com",
							Targets:    []string{"4.4.4.4"},
							RecordType: endpoint.RecordTypeA,
							RecordTTL:  endpoint.TTL(defaultTTL),
						},
						&endpoint.Endpoint{
							DNSName:    "www.beta.com",
							Targets:    []string{"ftp.beta.com"},
							RecordType: endpoint.RecordTypeCNAME,
							RecordTTL:  endpoint.TTL(defaultTTL),
							ProviderSpecific: endpoint.ProviderSpecific{
								{
									Name:  "hetzner-labels",
									Value: "env=production;project=beta.com",
								},
							},
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},

							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
						ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
							TTL: &testSecondTTL,
						},
					},
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
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
						recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "4.4.4.4",
								},
							},
						},
					},
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   2,
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Labels: map[string]string{
								"env": "test",
							},
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "ftp",
								},
							},
						},
						updateOpts: &hcloud.ZoneRRSetUpdateOpts{
							Labels: map[string]string{
								"env":     "production",
								"project": "beta.com",
							},
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

// Test_processDeleteActions tests processDeleteActions().
func Test_processDeleteActions(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneIDNameMapper zoneIDName
			rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
			deletesByZoneID  map[int64][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processDeleteActions(inp.zoneIDNameMapper, inp.rrSetsByZoneID, inp.deletesByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "No deletes created",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				deletesByZoneID  map[int64][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: zoneIDName{
					1: {
						ID:   1,
						Name: "alpha.com",
					},
					2: {
						ID:   2,
						Name: "beta.com",
					},
				},
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},

							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
					2: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
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
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Labels: map[string]string{
								"env": "test",
							},
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "ftp.beta.com",
								},
							},
						},
					},
				},
				deletesByZoneID: map[int64][]*endpoint.Endpoint{
					1: {
						&endpoint.Endpoint{
							DNSName:    "ccx.alpha.com",
							Targets:    endpoint.Targets{"7.7.7.7"},
							RecordType: "A",
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "deletes performed",
			input: struct {
				zoneIDNameMapper zoneIDName
				rrSetsByZoneID   map[int64][]*hcloud.ZoneRRSet
				deletesByZoneID  map[int64][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: zoneIDName{
					1: {
						ID:   1,
						Name: "alpha.com",
					},
					2: {
						ID:   2,
						Name: "beta.com",
					},
				},
				rrSetsByZoneID: map[int64][]*hcloud.ZoneRRSet{
					1: {
						&hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},

							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
					2: {
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
							},
							ID:   "id_2",
							Name: "ftp",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "2.2.2.2",
								},
								{
									Value: "4.4.4.4",
								},
							},
						},
						{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Labels: map[string]string{
								"env": "test",
							},
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "ftp.beta.com",
								},
							},
						},
					},
				},
				deletesByZoneID: map[int64][]*endpoint.Endpoint{
					1: {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"1.1.1.1"},
							RecordType: "A",
							RecordTTL:  endpoint.TTL(defaultTTL),
						},
					},
					2: {
						&endpoint.Endpoint{
							DNSName:    "www.beta.com",
							Targets:    endpoint.Targets{"ftp.beta.com"},
							RecordType: endpoint.RecordTypeCNAME,
							RecordTTL:  endpoint.TTL(defaultTTL),
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},

							ID:   "id_1",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							TTL:  &testFirstTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "beta.com",
							},
							ID:   "id_3",
							Name: "www",
							Type: hcloud.ZoneRRSetTypeCNAME,
							TTL:  &defaultTTL,
							Labels: map[string]string{
								"env": "test",
							},
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "ftp.beta.com",
								},
							},
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
