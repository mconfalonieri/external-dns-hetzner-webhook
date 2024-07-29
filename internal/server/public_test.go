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
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bsm/openmetrics"
	"github.com/stretchr/testify/assert"
)

// testPort is the test port for server on localhost.
const (
	testPort = 32128
	testHost = "localhost"
)

func Test_SetHealth(t *testing.T) {
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

func Test_SetReady(t *testing.T) {
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

func Test_IsHealthy(t *testing.T) {
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

func Test_IsReady(t *testing.T) {
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

func Test_livenessHandler(t *testing.T) {
	type testCase struct {
		name           string
		server         PublicServer
		expectedStatus int
		expectedText   string
	}
	testCases := []testCase{
		{
			name: "Server is alive",
			server: PublicServer{
				status: &HealthStatus{
					healthy: true,
				},
			},
			expectedStatus: http.StatusOK,
			expectedText:   http.StatusText(http.StatusOK),
		},
		{
			name: "Server is unhealthy",
			server: PublicServer{
				status: &HealthStatus{
					healthy: false,
				},
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedText:   http.StatusText(http.StatusServiceUnavailable),
		},
	}

	run := func(t *testing.T, tc testCase) {
		text := bytes.NewBuffer(make([]byte, 0))
		w := &httptest.ResponseRecorder{
			Body: text,
		}
		r := &http.Request{}
		tc.server.livenessHandler(w, r)
		assert.Equal(t, tc.expectedStatus, w.Code)
		assert.Equal(t, tc.expectedText, text.String())
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_readinessHandler(t *testing.T) {
	type testCase struct {
		name           string
		server         PublicServer
		expectedStatus int
		expectedText   string
	}
	testCases := []testCase{
		{
			name: "Server is ready",
			server: PublicServer{
				status: &HealthStatus{
					ready: true,
				},
			},
			expectedStatus: http.StatusOK,
			expectedText:   http.StatusText(http.StatusOK),
		},
		{
			name: "Server is not ready",
			server: PublicServer{
				status: &HealthStatus{
					ready: false,
				},
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedText:   http.StatusText(http.StatusServiceUnavailable),
		},
	}

	run := func(t *testing.T, tc testCase) {
		text := bytes.NewBuffer(make([]byte, 0))
		w := &httptest.ResponseRecorder{
			Body: text,
		}
		r := &http.Request{}
		tc.server.readinessHandler(w, r)
		assert.Equal(t, tc.expectedStatus, w.Code)
		assert.Equal(t, tc.expectedText, text.String())
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Start(t *testing.T) {
	status := &HealthStatus{
		healthy: true,
		ready:   true,
	}
	options := ServerOptions{
		HealthHost: testHost,
		HealthPort: testPort,
	}
	reg := &openmetrics.Registry{}

	startedChan := make(chan struct{})

	healthServer := PublicServer{
		status: status,
		reg:    reg,
	}

	go healthServer.Start(startedChan, options)
	<-startedChan

	url := fmt.Sprintf("http://%s:%d/ready", testHost, testPort)

	res, err := http.Get(url)

	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
}
