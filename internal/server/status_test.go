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

func Test_Status_SetHealth(t *testing.T) {
	type testCase struct {
		name     string
		instance *Status
		input    bool
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		obj.SetHealthy(tc.input)
		assert.Equal(t, tc.expected, obj.healthy.v)
	}

	testCases := []testCase{
		{
			name: "set to true",
			instance: &Status{
				healthy: mutexedBool{v: false},
			},
			input:    true,
			expected: true,
		},
		{
			name: "set to false",
			instance: &Status{
				healthy: mutexedBool{v: true},
			},
			input:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Status_SetReady(t *testing.T) {
	type testCase struct {
		name     string
		instance *Status
		input    bool
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		obj.SetReady(tc.input)
		assert.Equal(t, tc.expected, obj.ready.v)
	}

	testCases := []testCase{
		{
			name: "set to true",
			instance: &Status{
				ready: mutexedBool{v: false},
			},
			input:    true,
			expected: true,
		},
		{
			name: "set to false",
			instance: &Status{
				ready: mutexedBool{v: true},
			},
			input:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Status_IsHealthy(t *testing.T) {
	type testCase struct {
		name     string
		instance *Status
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		actual := obj.IsHealthy()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "not healthy",
			instance: &Status{
				healthy: mutexedBool{v: false},
			},
			expected: false,
		},
		{
			name: "healthy",
			instance: &Status{
				healthy: mutexedBool{v: true},
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

func Test_Status_IsReady(t *testing.T) {
	type testCase struct {
		name     string
		instance *Status
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		actual := obj.IsReady()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "not ready",
			instance: &Status{
				ready: mutexedBool{v: false},
			},
			expected: false,
		},
		{
			name: "ready",
			instance: &Status{
				ready: mutexedBool{v: true},
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
