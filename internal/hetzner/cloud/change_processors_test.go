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

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

var testZoneIDMapper = provider.ZoneIDName{
	"zoneIDAlpha": "alpha.com",
	"zoneIDBeta":  "beta.com",
}

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

// Test_processCreateActionsByZone tests processCreateActionsByZone().
func Test_processCreateActionsByZone(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneID    int64
			zoneName  string
			records   []hdns.Record
			endpoints []*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		changes := hetznerChanges{}
		processCreateActionsByZone(inp.zoneID, inp.zoneName, inp.records,
			inp.endpoints, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "record already created",
			input: struct {
				zoneID    string
				zoneName  string
				records   []hdns.Record
				endpoints []*endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				records: []hdns.Record{
					{
						Type:  "A",
						Name:  "www",
						Value: "127.0.0.1",
						Ttl:   7200,
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
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   &testTTL,
							Type:  "A",
							Value: "127.0.0.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
					},
				},
			},
		},
		{
			name: "new record created",
			input: struct {
				zoneID    string
				zoneName  string
				records   []hdns.Record
				endpoints []*endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				records: []hdns.Record{
					{
						Type:  "A",
						Name:  "ftp",
						Value: "127.0.0.1",
						Ttl:   7200,
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
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   &testTTL,
							Type:  "A",
							Value: "127.0.0.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
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
			zoneIDNameMapper provider.ZoneIDName
			recordsByZoneID  map[string][]hdns.Record
			createsByZoneID  map[string][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		changes := hetznerChanges{}
		processCreateActions(inp.zoneIDNameMapper, inp.recordsByZoneID,
			inp.createsByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				createsByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: testZoneIDMapper,
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							Type:  "A",
							Name:  "www",
							Value: "127.0.0.1",
						},
					},
				},
				createsByZoneID: map[string][]*endpoint.Endpoint{},
			},
		},
		{
			name: "empty changeset with key present",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				createsByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: testZoneIDMapper,
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							Type:  "A",
							Name:  "www",
							Value: "127.0.0.1",
						},
					},
				},
				createsByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {},
				},
			},
		},
		{
			name: "record already created",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				createsByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: testZoneIDMapper,
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							Type:  "A",
							Name:  "www",
							Value: "127.0.0.1",
							Ttl:   7200,
						},
					},
				},
				createsByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"127.0.0.1"},
							RecordType: "A",
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   &testTTL,
							Type:  "A",
							Value: "127.0.0.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
					},
				},
			},
		},
		{
			name: "new record created",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				createsByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: testZoneIDMapper,
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							Type:  "A",
							Name:  "ftp",
							Value: "127.0.0.1",
							Ttl:   7200,
						},
					},
				},
				createsByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"127.0.0.1"},
							RecordType: "A",
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   &testTTL,
							Type:  "A",
							Value: "127.0.0.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
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

// Test_processUpdateEndpoint tests processUpdateEndpoint().
func Test_processUpdateEndpoint(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneID                  string
			zoneName                string
			matchingRecordsByTarget map[string]hdns.Record
			ep                      *endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateEndpoint(inp.zoneID, inp.zoneName, inp.matchingRecordsByTarget,
			inp.ep, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "name changed",
			input: struct {
				zoneID                  string
				zoneName                string
				matchingRecordsByTarget map[string]hdns.Record
				ep                      *endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				matchingRecordsByTarget: map[string]hdns.Record{
					"1.1.1.1": {
						ID:   "id_1",
						Type: hdns.RecordTypeA,
						Name: "www",
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "1.1.1.1",
						Ttl:   -1,
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1"},
					RecordTTL:  -1,
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:   "id_1",
							Type: hdns.RecordTypeA,
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
						Options: &hdns.RecordUpdateOpts{
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl:   nil,
							Value: "1.1.1.1",
						},
					},
				},
			},
		},
		{
			name: "TTL changed",
			input: struct {
				zoneID                  string
				zoneName                string
				matchingRecordsByTarget map[string]hdns.Record
				ep                      *endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				matchingRecordsByTarget: map[string]hdns.Record{
					"1.1.1.1": {
						ID:   "id_1",
						Type: hdns.RecordTypeA,
						Name: "www",
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "1.1.1.1",
						Ttl:   -1,
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					RecordType: "A",
					Targets:    []string{"1.1.1.1"},
					RecordTTL:  7200,
				},
			},
			expectedChanges: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:   "id_1",
							Type: hdns.RecordTypeA,
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
						Options: &hdns.RecordUpdateOpts{
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl:   &testTTL,
							Value: "1.1.1.1",
						},
					},
				},
			},
		},
		{
			name: "target changed",
			input: struct {
				zoneID                  string
				zoneName                string
				matchingRecordsByTarget map[string]hdns.Record
				ep                      *endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				matchingRecordsByTarget: map[string]hdns.Record{
					"1.1.1.1": {
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
				},
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					RecordType: "A",
					Targets:    []string{"2.2.2.2"},
					RecordTTL:  -1,
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   nil,
							Type:  hdns.RecordTypeA,
							Value: "2.2.2.2",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
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

// Test_cleanupRemainingTargets tests cleanupRemainingTargets().
func Test_cleanupRemainingTargets(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneID                  string
			matchingRecordsByTarget map[string]hdns.Record
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		cleanupRemainingTargets(inp.zoneID, inp.matchingRecordsByTarget,
			&changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "no deletes",
			input: struct {
				zoneID                  string
				matchingRecordsByTarget map[string]hdns.Record
			}{
				zoneID:                  "zoneIDAlpha",
				matchingRecordsByTarget: map[string]hdns.Record{},
			},
		},
		{
			name: "delete",
			input: struct {
				zoneID                  string
				matchingRecordsByTarget map[string]hdns.Record
			}{
				zoneID: "zoneIDAlpha",
				matchingRecordsByTarget: map[string]hdns.Record{
					"1.1.1.1": {
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
				},
			},
			expectedChanges: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
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

// Test_getMatchingRecordsByTarget tests getMatchingRecordsByTarget().
func Test_getMatchingRecordsByTarget(t *testing.T) {
	type testCase struct {
		name     string
		input    []hdns.Record
		expected map[string]hdns.Record
	}

	run := func(t *testing.T, tc testCase) {
		actual := getMatchingRecordsByTarget(tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty array",
			input:    []hdns.Record{},
			expected: map[string]hdns.Record{},
		},
		{
			name: "some values",
			input: []hdns.Record{
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
					Type: hdns.RecordTypeA,
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					Value: "2.2.2.2",
					Ttl:   -1,
				},
			},
			expected: map[string]hdns.Record{
				"1.1.1.1": {
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
				"2.2.2.2": {
					ID:   "id_2",
					Name: "ftp",
					Type: hdns.RecordTypeA,
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					Value: "2.2.2.2",
					Ttl:   -1,
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
			zoneID    string
			zoneName  string
			records   []hdns.Record
			endpoints []*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateActionsByZone(inp.zoneID, inp.zoneName, inp.records,
			inp.endpoints, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zoneID    string
				zoneName  string
				records   []hdns.Record
				endpoints []*endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				records: []hdns.Record{
					{
						ID:   "id_1",
						Name: "www",
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "1.1.1.1",
						Ttl:   -1,
					},
				},
				endpoints: []*endpoint.Endpoint{},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "mixed changeset",
			input: struct {
				zoneID    string
				zoneName  string
				records   []hdns.Record
				endpoints []*endpoint.Endpoint
			}{
				zoneID:   "zoneIDAlpha",
				zoneName: "alpha.com",
				records: []hdns.Record{
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
						Type: hdns.RecordTypeA,
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "2.2.2.2",
						Ttl:   -1,
					},
				},
				endpoints: []*endpoint.Endpoint{
					{
						DNSName:    "www.alpha.com",
						RecordType: "A",
						Targets:    []string{"3.3.3.3"},
						RecordTTL:  -1,
					},
					{
						DNSName:    "ftp.alpha.com",
						RecordType: "A",
						Targets:    []string{"2.2.2.2"},
						RecordTTL:  7200,
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Ttl:   nil,
							Type:  hdns.RecordTypeA,
							Value: "3.3.3.3",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
					},
				},
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
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
					},
				},
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
						Options: &hdns.RecordUpdateOpts{
							Name: "ftp",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Type:  hdns.RecordTypeA,
							Value: "2.2.2.2",
							Ttl:   &testTTL,
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
			zoneIDNameMapper provider.ZoneIDName
			recordsByZoneID  map[string][]hdns.Record
			updatesByZoneID  map[string][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processUpdateActions(inp.zoneIDNameMapper, inp.recordsByZoneID,
			inp.updatesByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "empty changeset",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				updatesByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							ID:   "id_1",
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
					},
					"zoneIDBeta": {
						hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
					},
				},
				updatesByZoneID: map[string][]*endpoint.Endpoint{},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "empty changeset with key present",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				updatesByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							ID:   "id_1",
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
					},
					"zoneIDBeta": {
						hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
					},
				},
				updatesByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {},
					"zoneIDBeta":  {},
				},
			},
		},
		{
			name: "mixed changeset",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				updatesByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
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
					},
					"zoneIDBeta": {
						hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
					},
				},
				updatesByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							RecordType: "A",
							Targets:    []string{"3.3.3.3"},
							RecordTTL:  -1,
						},
					},
					"zoneIDBeta": {
						&endpoint.Endpoint{
							DNSName:    "ftp.beta.com",
							RecordType: "A",
							Targets:    []string{"2.2.2.2"},
							RecordTTL:  7200,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name:  "www",
							Type:  hdns.RecordTypeA,
							Value: "3.3.3.3",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: nil,
						},
					},
				},
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
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
					},
				},
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDBeta",
						Record: hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
						Options: &hdns.RecordUpdateOpts{
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   &testTTL,
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

func Test_targetsMatch(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			record hdns.Record
			ep     *endpoint.Endpoint
		}
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := targetsMatch(inp.record, inp.ep)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "record does not matches",
			input: struct {
				record hdns.Record
				ep     *endpoint.Endpoint
			}{
				record: hdns.Record{
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
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					Targets:    endpoint.Targets{"7.7.7.7"},
					RecordType: "A",
					RecordTTL:  -1,
				},
			},
			expected: false,
		},
		{
			name: "record matches",
			input: struct {
				record hdns.Record
				ep     *endpoint.Endpoint
			}{
				record: hdns.Record{
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
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					Targets:    endpoint.Targets{"1.1.1.1"},
					RecordType: "A",
					RecordTTL:  -1,
				},
			},
			expected: true,
		},
		{
			name: "cname special matching",
			input: struct {
				record hdns.Record
				ep     *endpoint.Endpoint
			}{
				record: hdns.Record{
					ID:   "id_2",
					Name: "ftp",
					Type: hdns.RecordTypeCNAME,
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
					Value: "www.beta.com.",
					Ttl:   -1,
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					Targets:    endpoint.Targets{"www.beta.com"},
					RecordType: "CNAME",
					RecordTTL:  -1,
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_processDeleteActionsByEndpoint tests processDeleteActionsByEndpoint().
func Test_processDeleteActionsByEndpoint(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			zoneID          string
			matchingRecords []hdns.Record
			ep              *endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processDeleteActionsByEndpoint(inp.zoneID, inp.matchingRecords,
			inp.ep, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "no matching records",
			input: struct {
				zoneID          string
				matchingRecords []hdns.Record
				ep              *endpoint.Endpoint
			}{
				zoneID:          "zoneIDAlpha",
				matchingRecords: []hdns.Record{},
				ep: &endpoint.Endpoint{
					DNSName:    "ccx.alpha.com",
					Targets:    endpoint.Targets{"7.7.7.7"},
					RecordType: "A",
					RecordTTL:  7200,
				},
			},
			expectedChanges: hetznerChanges{},
		},
		{
			name: "one matching record",
			input: struct {
				zoneID          string
				matchingRecords []hdns.Record
				ep              *endpoint.Endpoint
			}{
				zoneID: "zoneIDAlpha",
				matchingRecords: []hdns.Record{
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
						Name: "www",
						Type: hdns.RecordTypeA,
						Zone: &hdns.Zone{
							ID:   "zoneIDAlpha",
							Name: "alpha.com",
						},
						Value: "2.2.2.2",
						Ttl:   -1,
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "www.alpha.com",
					Targets:    endpoint.Targets{"1.1.1.1"},
					RecordType: "A",
					RecordTTL:  -1,
				},
			},
			expectedChanges: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
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
					},
				},
			},
		},
		{
			name: "cname special matching",
			input: struct {
				zoneID          string
				matchingRecords []hdns.Record
				ep              *endpoint.Endpoint
			}{
				zoneID: "zoneIDAlpha",
				matchingRecords: []hdns.Record{
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
						Value: "www.beta.com.",
						Ttl:   -1,
					},
				},
				ep: &endpoint.Endpoint{
					DNSName:    "ftp.alpha.com",
					Targets:    endpoint.Targets{"www.beta.com"},
					RecordType: "CNAME",
					RecordTTL:  -1,
				},
			},
			expectedChanges: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeCNAME,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "www.beta.com.",
							Ttl:   -1,
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
			zoneIDNameMapper provider.ZoneIDName
			recordsByZoneID  map[string][]hdns.Record
			deletesByZoneID  map[string][]*endpoint.Endpoint
		}
		expectedChanges hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		changes := hetznerChanges{}
		inp := tc.input
		processDeleteActions(inp.zoneIDNameMapper, inp.recordsByZoneID,
			inp.deletesByZoneID, &changes)
		assertEqualChanges(t, tc.expectedChanges, changes)
	}

	testCases := []testCase{
		{
			name: "No deletes created",
			input: struct {
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				deletesByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
							ID:   "id_1",
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "1.1.1.1",
							Ttl:   -1,
						},
					},
					"zoneIDBeta": {
						hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
					},
				},
				deletesByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {
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
				zoneIDNameMapper provider.ZoneIDName
				recordsByZoneID  map[string][]hdns.Record
				deletesByZoneID  map[string][]*endpoint.Endpoint
			}{
				zoneIDNameMapper: provider.ZoneIDName{
					"zoneIDAlpha": "alpha.com",
					"zoneIDBeta":  "beta.com",
				},
				recordsByZoneID: map[string][]hdns.Record{
					"zoneIDAlpha": {
						hdns.Record{
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
					},
					"zoneIDBeta": {
						hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
						hdns.Record{
							ID:   "id_3",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "4.4.4.4",
							Ttl:   -1,
						},
					},
				},
				deletesByZoneID: map[string][]*endpoint.Endpoint{
					"zoneIDAlpha": {
						&endpoint.Endpoint{
							DNSName:    "www.alpha.com",
							Targets:    endpoint.Targets{"1.1.1.1"},
							RecordType: "A",
							RecordTTL:  -1,
						},
					},
					"zoneIDBeta": {
						&endpoint.Endpoint{
							DNSName:    "ftp.beta.com",
							Targets:    endpoint.Targets{"2.2.2.2", "4.4.4.4"},
							RecordType: "A",
							RecordTTL:  -1,
						},
					},
				},
			},
			expectedChanges: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
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
					},
					{
						ZoneID: "zoneIDBeta",
						Record: hdns.Record{
							ID:   "id_2",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "2.2.2.2",
							Ttl:   -1,
						},
					},
					{
						ZoneID: "zoneIDBeta",
						Record: hdns.Record{
							ID:   "id_3",
							Name: "ftp",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDBeta",
								Name: "beta.com",
							},
							Value: "4.4.4.4",
							Ttl:   -1,
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
