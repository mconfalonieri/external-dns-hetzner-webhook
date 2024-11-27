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

func testHandlerArgs() (*httptest.ResponseRecorder, *http.Request) {
	text := bytes.NewBuffer(make([]byte, 0))
	w := &httptest.ResponseRecorder{Body: text}
	r := &http.Request{}
	return w, r
}

func Test_MetricsSocket_livenessHandler(t *testing.T) {
	type testCase struct {
		name     string
		instance *MetricsSocket
		expected struct {
			status int
			text   string
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		exp := tc.expected
		w, r := testHandlerArgs()
		obj.livenessHandler(w, r)
		assert.Equal(t, exp.status, w.Code)
		assert.Equal(t, exp.text, w.Body.String())
	}

	testCases := []testCase{
		{
			name: "server is healthy",
			instance: &MetricsSocket{
				status: &Status{
					healthy: mutexedBool{v: true},
				},
			},
			expected: struct {
				status int
				text   string
			}{
				status: http.StatusOK,
				text:   http.StatusText(http.StatusOK),
			},
		},
		{
			name: "server is not healthy",
			instance: &MetricsSocket{
				status: &Status{
					healthy: mutexedBool{v: false},
				},
			},
			expected: struct {
				status int
				text   string
			}{
				status: http.StatusServiceUnavailable,
				text:   http.StatusText(http.StatusServiceUnavailable),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_MetricsSocket_readinessHandler(t *testing.T) {
	type testCase struct {
		name     string
		instance *MetricsSocket
		expected struct {
			status int
			text   string
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.instance
		exp := tc.expected
		w, r := testHandlerArgs()
		obj.readinessHandler(w, r)
		assert.Equal(t, exp.status, w.Code)
		assert.Equal(t, exp.text, w.Body.String())
	}

	testCases := []testCase{
		{
			name: "server is ready",
			instance: &MetricsSocket{
				status: &Status{
					ready: mutexedBool{v: true},
				},
			},
			expected: struct {
				status int
				text   string
			}{
				status: http.StatusOK,
				text:   http.StatusText(http.StatusOK),
			},
		},
		{
			name: "server is not ready",
			instance: &MetricsSocket{
				status: &Status{
					ready: mutexedBool{v: false},
				},
			},
			expected: struct {
				status int
				text   string
			}{
				status: http.StatusServiceUnavailable,
				text:   http.StatusText(http.StatusServiceUnavailable),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Start(t *testing.T) {
	status := &Status{
		healthy: mutexedBool{v: true},
		ready:   mutexedBool{v: true},
	}
	options := SocketOptions{
		MetricsHost: testHost,
		MetricsPort: testPort,
	}
	reg := &openmetrics.Registry{}

	startedChan := make(chan struct{})

	metricsSocket := MetricsSocket{
		status: status,
		reg:    reg,
	}

	go metricsSocket.Start(startedChan, options)
	<-startedChan

	url := fmt.Sprintf("http://%s:%d/ready", testHost, testPort)

	res, err := http.Get(url)

	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
}
