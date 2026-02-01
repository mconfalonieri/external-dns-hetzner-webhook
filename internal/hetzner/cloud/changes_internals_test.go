/*
 * Changes Internals - unit tests
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
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// changeType is used to test the GetLogFields method.
type changeType interface {
	GetLogFields() log.Fields
}

var defaultTTL = 7200

// getRRSetRecordsString represents a recordset as a string.
func Test_getRRSetRecordsString(t *testing.T) {
	type testCase struct {
		name     string
		input    []hcloud.ZoneRRSetRecord
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := getRRSetRecordsString(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty recordset",
			input:    []hcloud.ZoneRRSetRecord{},
			expected: "",
		},
		{
			name: "some records",
			input: []hcloud.ZoneRRSetRecord{
				{Value: "1.1.1.1"},
				{Value: "2.2.2.2"},
				{Value: "3.3.3.3"},
			},
			expected: "1.1.1.1;2.2.2.2;3.3.3.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_GetLogFields(t *testing.T) {
	type testCase struct {
		name     string
		object   changeType
		expected log.Fields
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.object.GetLogFields()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "hetznerChangeCreate",
			object: &hetznerChangeCreate{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
				opts: hcloud.ZoneRRSetCreateOpts{
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "1.1.1.1",
						},
						{
							Value: "2.2.2.2",
						},
					},
					TTL: &testTTL,
				},
			},
			expected: log.Fields{
				"zone":       "alpha.com",
				"dnsName":    "www",
				"recordType": "A",
				"targets":    "1.1.1.1;2.2.2.2",
				"ttl":        "7200",
			},
		},
		{
			name: "hetznerChangeUpdate",
			object: &hetznerChangeUpdate{
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
			expected: log.Fields{
				"zone":       "alpha.com",
				"dnsName":    "www",
				"recordType": "A",
				"*targets":   "1.1.1.1;3.3.3.3",
				"*ttl":       "20000",
				"*labels":    "testLabel=testValue",
			},
		},
		{
			name: "hetznerChangeDelete",
			object: &hetznerChangeDelete{
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
			expected: log.Fields{
				"zone":       "alpha.com",
				"dnsName":    "www",
				"recordType": "A",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
