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
package hetzner

import (
	"context"
	"errors"
	"testing"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/stretchr/testify/assert"
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
			name: "Creations",
			changes: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID:  "alphaZoneID",
						Options: &hdns.RecordCreateOpts{},
					},
				},
			},
		},
		{
			name: "Updates",
			changes: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID:  "alphaZoneID",
						Record:  hdns.Record{},
						Options: &hdns.RecordUpdateOpts{},
					},
				},
			},
		},
		{
			name: "Deletions",
			changes: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "alphaZoneID",
						Record: hdns.Record{},
					},
				},
			},
		},
		{
			name: "All",
			changes: hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID:  "alphaZoneID",
						Options: &hdns.RecordCreateOpts{},
					},
				},
				updates: []*hetznerChangeUpdate{
					{
						ZoneID:  "alphaZoneID",
						Record:  hdns.Record{},
						Options: &hdns.RecordUpdateOpts{},
					},
				},
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "alphaZoneID",
						Record: hdns.Record{},
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

// Test_hetznerChanges_AddChangeCreate tests hetznerChanges.AddChangeCreate().
func Test_hetznerChanges_AddChangeCreate(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    struct {
			zoneID  string
			options *hdns.RecordCreateOpts
		}
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := tc.instance
		actual.AddChangeCreate(inp.zoneID, inp.options)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add create",
			instance: hetznerChanges{},
			input: struct {
				zoneID  string
				options *hdns.RecordCreateOpts
			}{
				zoneID: "zoneIDAlpha",
				options: &hdns.RecordCreateOpts{
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
			expected: hetznerChanges{
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

// Test_hetznerChanges_AddChangeUpdate tests hetznerChanges.AddChangeUpdate().
func Test_hetznerChanges_AddChangeUpdate(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    struct {
			zoneID  string
			record  hdns.Record
			options *hdns.RecordUpdateOpts
		}
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := tc.instance
		actual.AddChangeUpdate(inp.zoneID, inp.record, inp.options)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add update",
			instance: hetznerChanges{},
			input: struct {
				zoneID  string
				record  hdns.Record
				options *hdns.RecordUpdateOpts
			}{
				zoneID: "zoneIDAlpha",
				record: hdns.Record{
					ID:    "id_1",
					Name:  "www",
					Ttl:   -1,
					Type:  "A",
					Value: "127.0.0.1",
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
				},
				options: &hdns.RecordUpdateOpts{
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
			expected: hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id_1",
							Name:  "www",
							Ttl:   -1,
							Type:  "A",
							Value: "127.0.0.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
						},
						Options: &hdns.RecordUpdateOpts{
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

// addChangeDelete adds a new delete entry to the current object.
func Test_hetznerChanges_AddChangeDelete(t *testing.T) {
	type testCase struct {
		name     string
		instance hetznerChanges
		input    struct {
			zoneID string
			record hdns.Record
		}
		expected hetznerChanges
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := tc.instance
		actual.AddChangeDelete(inp.zoneID, inp.record)
		assert.EqualValues(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "add update",
			instance: hetznerChanges{},
			input: struct {
				zoneID string
				record hdns.Record
			}{
				zoneID: "zoneIDAlpha",
				record: hdns.Record{
					ID:    "id_1",
					Name:  "www",
					Ttl:   -1,
					Type:  "A",
					Value: "127.0.0.1",
					Zone: &hdns.Zone{
						ID:   "zoneIDAlpha",
						Name: "alpha.com",
					},
				},
			},
			expected: hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id_1",
							Name:  "www",
							Ttl:   -1,
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

// applyDeletes processes the records to be deleted.
func Test_hetznerChanges_applyDeletes(t *testing.T) {
	type testCase struct {
		name          string
		changes       *hetznerChanges
		input         *mockClient
		expectedState mockClientState
		expectedErr   bool
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.changes.applyDeletes(context.Background(), tc.input)
		checkError(t, err, tc.expectedErr)
		assert.Equal(t, tc.expectedState, tc.input.GetState())
	}

	testCases := []testCase{
		{
			name: "deletion",
			changes: &hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id1",
							Type:  hdns.RecordTypeA,
							Name:  "www",
							Value: "1.1.1.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: -1,
						},
					},
				},
			},
			input:         &mockClient{},
			expectedState: mockClientState{DeleteRecordCalled: true},
		},
		{
			name: "deletion error",
			changes: &hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id1",
							Type:  hdns.RecordTypeA,
							Name:  "www",
							Value: "1.1.1.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: -1,
						},
					},
				},
			},
			input: &mockClient{
				deleteRecord: deleteResponse{
					err: errors.New("test delete error"),
				},
			},
			expectedState: mockClientState{DeleteRecordCalled: true},
			expectedErr:   true,
		},
		{
			name: "deletion dry run",
			changes: &hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id1",
							Type:  hdns.RecordTypeA,
							Name:  "www",
							Value: "1.1.1.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: -1,
						},
					},
				},
				dryRun: true,
			},
			input:         &mockClient{},
			expectedState: mockClientState{},
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
		name          string
		changes       *hetznerChanges
		input         *mockClient
		expectedState mockClientState
		expectedErr   bool
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.changes.applyCreates(context.Background(), tc.input)
		checkError(t, err, tc.expectedErr)
		assert.Equal(t, tc.expectedState, tc.input.GetState())
	}

	testCases := []testCase{
		{
			name: "creation",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name: "www",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			input:         &mockClient{},
			expectedState: mockClientState{CreateRecordCalled: true},
		},
		{
			name: "creation error",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name: "www",
							Type: hdns.RecordTypeA,
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			input: &mockClient{
				createRecord: recordResponse{
					err: errors.New("test creation error"),
				},
			},
			expectedState: mockClientState{CreateRecordCalled: true},
			expectedErr:   true,
		},
		{
			name: "creation dry run",
			input: &mockClient{
				createRecord: recordResponse{
					err: errors.New("test creation error"),
				},
			},
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
				dryRun: true,
			},
			expectedState: mockClientState{},
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
		name          string
		changes       *hetznerChanges
		input         *mockClient
		expectedState mockClientState
		expectedErr   bool
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.changes.applyUpdates(context.Background(), tc.input)
		checkError(t, err, tc.expectedErr)
		assert.Equal(t, tc.expectedState, tc.input.GetState())
	}

	testCases := []testCase{
		{
			name:  "update",
			input: &mockClient{},
			changes: &hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "www",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   testTTL,
						},
						Options: &hdns.RecordUpdateOpts{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "ftp",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			expectedState: mockClientState{UpdateRecordCalled: true},
		},
		{
			name: "update error",
			input: &mockClient{
				updateRecord: recordResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "www",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   testTTL,
						},
						Options: &hdns.RecordUpdateOpts{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "ftp",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			expectedState: mockClientState{UpdateRecordCalled: true},
			expectedErr:   true,
		},
		{
			name:  "update dry run",
			input: &mockClient{},
			changes: &hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "www",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   testTTL,
						},
						Options: &hdns.RecordUpdateOpts{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "ftp",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
				dryRun: true,
			},
			expectedState: mockClientState{},
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
		name          string
		changes       *hetznerChanges
		input         *mockClient
		expectedState mockClientState
		expectedErr   bool
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.changes.ApplyChanges(context.Background(), tc.input)
		checkError(t, err, tc.expectedErr)
		assert.Equal(t, tc.expectedState, tc.input.GetState())
	}

	testCases := []testCase{
		{
			name:          "no changes",
			changes:       &hetznerChanges{},
			input:         &mockClient{},
			expectedState: mockClientState{},
			expectedErr:   false,
		},
		{
			name: "all changes",
			changes: &hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id1",
							Type:  hdns.RecordTypeA,
							Name:  "www",
							Value: "1.1.1.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: -1,
						},
					},
				},
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "www",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   testTTL,
						},
						Options: &hdns.RecordUpdateOpts{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "ftp",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			input: &mockClient{},
			expectedState: mockClientState{
				CreateRecordCalled: true,
				DeleteRecordCalled: true,
				UpdateRecordCalled: true,
			},
		},
		{
			name: "deletion error",
			changes: &hetznerChanges{
				deletes: []*hetznerChangeDelete{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							ID:    "id1",
							Type:  hdns.RecordTypeA,
							Name:  "www",
							Value: "1.1.1.1",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Ttl: -1,
						},
					},
				},
			},
			input: &mockClient{
				deleteRecord: deleteResponse{
					err: errors.New("test delete error"),
				},
			},
			expectedState: mockClientState{DeleteRecordCalled: true},
			expectedErr:   true,
		},
		{
			name: "creation error",
			changes: &hetznerChanges{
				creates: []*hetznerChangeCreate{
					{
						ZoneID: "zoneIDAlpha",
						Options: &hdns.RecordCreateOpts{
							Name: "www",
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			input: &mockClient{
				createRecord: recordResponse{
					err: errors.New("test creation error"),
				},
			},
			expectedState: mockClientState{CreateRecordCalled: true},
			expectedErr:   true,
		},
		{
			name: "update error",
			input: &mockClient{
				updateRecord: recordResponse{
					err: errors.New("test update error"),
				},
			},
			changes: &hetznerChanges{
				updates: []*hetznerChangeUpdate{
					{
						ZoneID: "zoneIDAlpha",
						Record: hdns.Record{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "www",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   testTTL,
						},
						Options: &hdns.RecordUpdateOpts{
							Zone: &hdns.Zone{
								ID:   "zoneIDAlpha",
								Name: "alpha.com",
							},
							Name:  "ftp",
							Type:  "A",
							Value: "127.0.0.1",
							Ttl:   &testTTL,
						},
					},
				},
			},
			expectedState: mockClientState{UpdateRecordCalled: true},
			expectedErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
