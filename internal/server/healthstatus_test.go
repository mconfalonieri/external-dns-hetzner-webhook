/*
 * Copyright 2023 Marco Confalonieri.
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
package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_HealthStatus_SetHealth(t *testing.T) {
	type testCase struct {
		name     string
		status   *HealthStatus
		input    bool
		expected bool
	}
	testCases := []testCase{
		{
			name:     "Set to true",
			status:   &HealthStatus{healthy: false},
			input:    true,
			expected: true,
		},
		{
			name:     "Set to false",
			status:   &HealthStatus{healthy: true},
			input:    false,
			expected: false,
		},
	}
	run := func(t *testing.T, tc testCase) {
		tc.status.SetHealth(tc.input)
		assert.Equal(t, tc.expected, tc.status.healthy)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_HealthStatus_SetReady(t *testing.T) {
	type testCase struct {
		name     string
		status   *HealthStatus
		input    bool
		expected bool
	}
	testCases := []testCase{
		{
			name:     "Set to true",
			status:   &HealthStatus{ready: false},
			input:    true,
			expected: true,
		},
		{
			name:     "Set to false",
			status:   &HealthStatus{ready: true},
			input:    false,
			expected: false,
		},
	}
	run := func(t *testing.T, tc testCase) {
		tc.status.SetReady(tc.input)
		assert.Equal(t, tc.expected, tc.status.ready)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_HealthStatus_IsHealthy(t *testing.T) {
	type testCase struct {
		name     string
		status   *HealthStatus
		expected bool
	}
	testCases := []testCase{
		{
			name:     "Status is not healthy",
			status:   &HealthStatus{healthy: false},
			expected: false,
		},
		{
			name:     "Status is healthy",
			status:   &HealthStatus{healthy: true},
			expected: true,
		},
	}
	run := func(t *testing.T, tc testCase) {
		actual := tc.status.IsHealthy()
		assert.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_HealthStatus_IsReady(t *testing.T) {
	type testCase struct {
		name     string
		status   *HealthStatus
		input    bool
		expected bool
	}
	testCases := []testCase{
		{
			name:     "Set to true",
			status:   &HealthStatus{ready: false},
			input:    true,
			expected: true,
		},
		{
			name:     "Set to false",
			status:   &HealthStatus{ready: true},
			input:    false,
			expected: false,
		},
	}
	run := func(t *testing.T, tc testCase) {
		tc.status.SetReady(tc.input)
		assert.Equal(t, tc.expected, tc.status.ready)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
