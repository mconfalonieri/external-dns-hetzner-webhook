/*
 * ZoneIDMap - Unit Tests.
 *
 * Copyright 2025 Marco Confalonieri.
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

// Test_zoneIDName_Add tests zoneIDName.Add().
func Test_zoneIDName_Add(t *testing.T) {
	type testCase struct {
		name           string
		object         zoneIDName
		input          *hcloud.Zone
		expectedObject zoneIDName
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		obj.Add(tc.input)
		assert.Equal(t, tc.expectedObject, obj)
	}

	testCases := []testCase{
		{
			name:           "nil zone",
			object:         zoneIDName{},
			input:          nil,
			expectedObject: zoneIDName{},
		},
		{
			name:   "add zone",
			object: zoneIDName{},
			input: &hcloud.Zone{
				ID:   1,
				Name: "alpha.com",
			},
			expectedObject: zoneIDName{
				1: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
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

// Test_zoneIDName_FindZone tests zoneIDName.FindZone().
func Test_zoneIDName_FindZone(t *testing.T) {
	type testCase struct {
		name     string
		object   zoneIDName
		input    string
		expected struct {
			zoneID int64
			zone   *hcloud.Zone
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		exp := tc.expected
		zoneID, zone := obj.FindZone(tc.input)
		assert.Equal(t, exp.zoneID, zoneID)
		assert.Equal(t, exp.zone, zone)
	}

	testCases := []testCase{
		{
			name: "zone found 1",
			object: zoneIDName{
				1: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
			},
			input: "www.alpha.com",
			expected: struct {
				zoneID int64
				zone   *hcloud.Zone
			}{
				zoneID: 1,
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
			},
		},
		{
			name: "zone found 2",
			object: zoneIDName{
				1: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
			},
			input: "www.sub.alpha.com",
			expected: struct {
				zoneID int64
				zone   *hcloud.Zone
			}{
				zoneID: 1,
				zone: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
			},
		},
		{
			name: "zone not found",
			object: zoneIDName{
				1: &hcloud.Zone{
					ID:   1,
					Name: "alpha.com",
				},
			},
			input: "www.beta.com",
			expected: struct {
				zoneID int64
				zone   *hcloud.Zone
			}{
				zoneID: -1,
				zone:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
