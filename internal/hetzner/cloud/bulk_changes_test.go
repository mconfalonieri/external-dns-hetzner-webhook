/*
 * Bulk changes - unit tests.
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
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

var (
	ttl3600 = 3600
	ttl7200 = 7200
)

// Test_bulkChanges_empty tests hetznerChanges.empty().
func Test_bulkChanges_empty(t *testing.T) {
	type testCase struct {
		name     string
		object   bulkChanges
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.empty()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty",
			object:   bulkChanges{},
			expected: true,
		},
		{
			name: "not empty",
			object: bulkChanges{
				changes: map[int64]*zoneChanges{
					1: {},
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

func Test_bulkChanges_getZoneChanges(t *testing.T) {
	type testCase struct {
		name      string
		object    bulkChanges
		input     *hcloud.Zone
		expected  *zoneChanges
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.getZoneChanges(tc.input)
		assert.Equal(t, tc.expected, actual)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "not present",
			object: bulkChanges{
				zones:   map[int64]*hcloud.Zone{},
				changes: map[int64]*zoneChanges{},
			},
			input: &hcloud.Zone{ID: 1, Name: "alpha.com"},
			expected: &zoneChanges{
				creates: []*hetznerChangeCreate{},
				updates: []*hetznerChangeUpdate{},
				deletes: []*hetznerChangeDelete{},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
		},
		{
			name: "present",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: &hcloud.Zone{ID: 1, Name: "alpha.com"},
			expected: &zoneChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "10.0.0.1",
								},
							},
							TTL: &ttl3600,
						},
					},
				},
				updates: []*hetznerChangeUpdate{},
				deletes: []*hetznerChangeDelete{},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
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

func Test_bulkChanges_AddChangeCreate(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			opts hcloud.ZoneRRSetCreateOpts
		}
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.AddChangeCreate(inp.zone, inp.opts)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add create",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				opts hcloud.ZoneRRSetCreateOpts
			}{
				zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
				opts: hcloud.ZoneRRSetCreateOpts{
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
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

func Test_bulkChanges_AddChangeUpdate(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			rrset       *hcloud.ZoneRRSet
			ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
			recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
			updateOpts  *hcloud.ZoneRRSetUpdateOpts
		}
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.AddChangeUpdate(inp.rrset, inp.ttlOpts, inp.recordsOpts, inp.updateOpts)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add ttl update",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				rrset       *hcloud.ZoneRRSet
				ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
				recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
				updateOpts  *hcloud.ZoneRRSetUpdateOpts
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
				ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &ttl7200,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
		},
		{
			name: "add ttl update",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				rrset       *hcloud.ZoneRRSet
				ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
				recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
				updateOpts  *hcloud.ZoneRRSetUpdateOpts
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
				ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &ttl7200,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
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
