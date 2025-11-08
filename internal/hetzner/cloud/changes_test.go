/*
 * Changes - unit tests.
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
	"context"
	"errors"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

// Test TTLs for update
var (
	testFirstTTL  = 10000
	testSecondTTL = 20000
)

// Test_hetznerChanges_empty tests hetznerChanges.empty().
func Test_hetznerChanges_empty(t *testing.T) {
	type testCase struct {
		name     string
		changes  hetznerChanges
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.changes.empty()
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "Empty",
			changes:  hetznerChanges{},
			expected: true,
		},
		{
			name: "Creations present",
			changes: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{},
				},
			},
			expected: false,
		},
		{
			name: "Updates present",
			changes: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{},
				},
			},
			expected: false,
		},
		{
			name: "Deletions present",
			changes: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{},
				},
			},
			expected: false,
		},
		{
			name: "All present",
			changes: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{},
				},
				updates: []*hetznerChangeUpdate{
					{},
				},
				deletes: []*hetznerChangeDelete{
					{},
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

// Test_hetznerChanges_AddChangeCreate tests hetznerChanges.AddChangeCreate().
func Test_hetznerChanges_AddChangeCreate(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    struct {
			zone *hcloud.Zone
			opts hcloud.ZoneRRSetCreateOpts
		}
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := tc.instance
		actual.AddChangeCreate(inp.zone, inp.opts)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add create",
			instance: hetznerChanges{},
			input: struct {
				zone *hcloud.Zone
				opts hcloud.ZoneRRSetCreateOpts
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
				opts: hcloud.ZoneRRSetCreateOpts{
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "127.0.0.1",
						},
					},
					TTL: &testTTL,
				},
			},
			expected: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
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

// Test_hetznerChanges_AddChangeUpdate tests hetznerChanges.AddChangeUpdate().
func Test_hetznerChanges_AddChangeUpdate(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    struct {
			rrset       *hcloud.ZoneRRSet
			ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
			recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
			updateOpts  *hcloud.ZoneRRSetUpdateOpts
		}
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := tc.instance
		actual.AddChangeUpdate(inp.rrset, inp.ttlOpts, inp.recordsOpts, inp.updateOpts)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add update",
			instance: hetznerChanges{},
			input: struct {
				rrset       *hcloud.ZoneRRSet
				ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
				recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
				updateOpts  *hcloud.ZoneRRSetUpdateOpts
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{
						ID:   1,
						Name: "alpha.com",
					},
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
				},
				ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &testTTL,
				},
			},
			expected: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						rrset: &hcloud.ZoneRRSet{
							Zone: &hcloud.Zone{
								ID:   1,
								Name: "alpha.com",
							},
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
						},
						ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
							TTL: &testTTL,
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

// addChangeDelete adds a new delete entry to the current object.
func Test_hetznerChanges_AddChangeDelete(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    *hcloud.ZoneRRSet
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.instance
		actual.AddChangeDelete(tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add delete",
			instance: hetznerChanges{},
			input: &hcloud.ZoneRRSet{
				Zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
				ID:   "id_1",
				Name: "www",
				Type: hcloud.ZoneRRSetTypeA,
				TTL:  &testTTL,
				Records: []hcloud.ZoneRRSetRecord{
					{
						Value: "1.1.1.1",
					},
				},
			},
			expected: hetznerChanges{
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
							TTL:  &testTTL,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// applyDeletes processes the records to be deleted.
func Test_hetznerChanges_applyDeletes(t *testing.T) {
	type testCase struct {
		name     string
		changes  *hetznerChanges
		input    *mockClient
		expected struct {
			state mockClientState
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		err := tc.changes.applyDeletes(context.Background(), inp)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.state, inp.GetState())
	}

	testCases := []testCase{
		{
			name: "deletion",
			changes: &hetznerChanges{
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
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
			},
			input: &mockClient{},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{DeleteRRSetCalled: true},
			},
		},
		{
			name: "deletion error",
			changes: &hetznerChanges{
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
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
			},
			input: &mockClient{
				deleteRRSet: deleteRRSetResponse{
					err: errors.New("test delete error"),
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{DeleteRRSetCalled: true},
				err:   errors.New("test delete error"),
			},
		},
		{
			name: "deletion dry run",
			changes: &hetznerChanges{
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
							TTL:  &defaultTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
				dryRun: true,
			},
			input: &mockClient{},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// applyCreates processes the records to be created.
func Test_hetznerChanges_applyCreates(t *testing.T) {
	type testCase struct {
		name     string
		changes  *hetznerChanges
		input    *mockClient
		expected struct {
			state mockClientState
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		err := tc.changes.applyCreates(context.Background(), inp)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.state, inp.GetState())
	}

	testCases := []testCase{
		{
			name: "creation",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
							Labels: map[string]string{
								"env": "test",
							},
						},
					},
				},
			},
			input: &mockClient{},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{CreateRRSetCalled: true},
			},
		},
		{
			name: "creation error",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
						},
					},
				},
			},
			input: &mockClient{
				createRRSet: createRRSetResponse{
					err: errors.New("test creation error"),
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{CreateRRSetCalled: true},
				err:   errors.New("test creation error"),
			},
		},
		{
			name: "creation dry run",
			input: &mockClient{
				createRRSet: createRRSetResponse{
					err: errors.New("test creation error"),
				},
			},
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
						},
					},
				},
				dryRun: true,
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// applyUpdates processes the records to be updated.
func Test_hetznerChanges_applyUpdates(t *testing.T) {
	type testCase struct {
		name     string
		changes  *hetznerChanges
		input    *mockClient
		expected struct {
			state mockClientState
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		err := tc.changes.applyUpdates(context.Background(), inp)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.state, inp.GetState())
	}

	testCases := []testCase{
		{
			name:  "update TTL",
			input: &mockClient{},
			changes: &hetznerChanges{
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
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					UpdateRRSetTTLCalled: true,
				},
			},
		},
		{
			name:  "update records",
			input: &mockClient{},
			changes: &hetznerChanges{
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
								{
									Value: "3.3.3.3",
								},
							},
						},
					},
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{UpdateRRSetRecordsCalled: true},
			},
		},
		{
			name:  "update all",
			input: &mockClient{},
			changes: &hetznerChanges{
				slash: "--slash--",
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
								"testLabel": "testValue",
							},
						},
					},
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					UpdateRRSetTTLCalled:     true,
					UpdateRRSetRecordsCalled: true,
					UpdateRRSetLabelsCalled:  true,
				},
			},
		},
		{
			name: "update TTL error",
			input: &mockClient{
				updateRRSetTTL: actionResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
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
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{UpdateRRSetTTLCalled: true},
				err:   errors.New("test update error"),
			},
		},
		{
			name: "update records error",
			input: &mockClient{
				updateRRSetRecords: actionResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
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
								{
									Value: "3.3.3.3",
								},
							},
						},
					},
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{UpdateRRSetRecordsCalled: true},
				err:   errors.New("test update error"),
			},
		},
		{
			name:  "update dry run",
			input: &mockClient{},
			changes: &hetznerChanges{
				slash: "--slash--",
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
								"testLabel": "testValue",
							},
						},
					},
				},
				dryRun: true,
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_hetznerChanges_ApplyChanges tests hetznerChanges.ApplyChanges().
func Test_hetznerChanges_ApplyChanges(t *testing.T) {
	type testCase struct {
		name     string
		changes  *hetznerChanges
		input    *mockClient
		expected struct {
			state mockClientState
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		err := tc.changes.ApplyChanges(context.Background(), inp)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.state, inp.GetState())
	}

	testCases := []testCase{
		{
			name:    "no changes",
			changes: &hetznerChanges{},
			input:   &mockClient{},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{},
			},
		},
		{
			name: "all changes",
			changes: &hetznerChanges{
				slash: "--slash--",
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
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
						},
					},
				},
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
								"testLabel": "testValue",
							},
						},
					},
				},
			},
			input: &mockClient{},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					CreateRRSetCalled:        true,
					UpdateRRSetTTLCalled:     true,
					UpdateRRSetRecordsCalled: true,
					UpdateRRSetLabelsCalled:  true,
					DeleteRRSetCalled:        true,
				},
			},
		},
		{
			name: "deletion error",
			changes: &hetznerChanges{
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
							TTL:  &testTTL,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "1.1.1.1",
								},
							},
						},
					},
				},
			},
			input: &mockClient{
				deleteRRSet: deleteRRSetResponse{
					err: errors.New("test delete error"),
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					DeleteRRSetCalled: true,
				},
				err: errors.New("test delete error"),
			},
		},
		{
			name: "creation error",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{
							ID:   1,
							Name: "alpha.com",
						},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "127.0.0.1",
								},
							},
							TTL: &testTTL,
						},
					},
				},
			},
			input: &mockClient{
				createRRSet: createRRSetResponse{
					err: errors.New("test creation error"),
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					CreateRRSetCalled: true,
				},
				err: errors.New("test creation error"),
			},
		},
		{
			name: "update TTL error",
			input: &mockClient{
				updateRRSetTTL: actionResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
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
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					UpdateRRSetTTLCalled: true,
				},
				err: errors.New("test update error"),
			},
		},
		{
			name: "update records error",
			input: &mockClient{
				updateRRSetRecords: actionResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
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
								{
									Value: "3.3.3.3",
								},
							},
						},
					},
				},
			},
			expected: struct {
				state mockClientState
				err   error
			}{
				state: mockClientState{
					UpdateRRSetRecordsCalled: true,
				},
				err: errors.New("test update error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
