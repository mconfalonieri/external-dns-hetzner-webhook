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
	"time"

	"github.com/codingconcepts/env"
	"github.com/stretchr/testify/assert"
)

func Test_SocketOptions_defaults(t *testing.T) {
	actual := SocketOptions{}
	expected := SocketOptions{
		WebhookHost:  "localhost",
		WebhookPort:  uint16(8888),
		MetricsHost:  "0.0.0.0",
		MetricsPort:  uint16(8080),
		ReadTimeout:  60000,
		WriteTimeout: 60000,
	}

	// Assign the default values.
	if err := env.Set(&actual); err != nil {
		t.Fail()
	}

	assert.Equal(t, expected, actual)
}

func Test_SocketOptions_GetWebhookAddress(t *testing.T) {
	type testCase struct {
		name     string
		options  SocketOptions
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.options
		actual := obj.GetWebhookAddress()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "webhook address with ip",
			options: SocketOptions{
				WebhookHost: "10.0.0.1",
				WebhookPort: 1000,
			},
			expected: "10.0.0.1:1000",
		},
		{
			name: "webhook address with hostname",
			options: SocketOptions{
				WebhookHost: "localhost",
				WebhookPort: 8888,
			},
			expected: "localhost:8888",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_SocketOptions_GetMetricsAddress(t *testing.T) {
	type testCase struct {
		name     string
		options  SocketOptions
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.options
		actual := obj.GetMetricsAddress()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "metrics address with ip",
			options: SocketOptions{
				MetricsHost: "10.0.0.2",
				MetricsPort: 2000,
			},
			expected: "10.0.0.2:2000",
		},
		{
			name: "metrics address with hostname",
			options: SocketOptions{
				MetricsHost: "broadcast",
				MetricsPort: 8080,
			},
			expected: "broadcast:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_SocketOptions_timeouts(t *testing.T) {
	const testReadTimeout = time.Duration(5000) * time.Millisecond
	const testWriteTimeout = time.Duration(15000) * time.Millisecond
	s := SocketOptions{
		ReadTimeout:  5000,
		WriteTimeout: 15000,
	}

	r := s.GetReadTimeout()
	w := s.GetWriteTimeout()

	assert.Equal(t, r, testReadTimeout)
	assert.Equal(t, w, testWriteTimeout)
}
