/*
 * Main - unit tests
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
package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner"
	hetznercloud "external-dns-hetzner-webhook/internal/hetzner/cloud"
	hetznerdns "external-dns-hetzner-webhook/internal/hetzner/dns"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/provider"
)

// mockStatus provides a test status object.
type mockStatus struct {
	ready   bool
	healthy bool
}

// SetHealthy sets the health status flag.
func (s *mockStatus) SetHealthy(health bool) {
	s.healthy = health
}

// SetReady sets the readiness flag.
func (s *mockStatus) SetReady(ready bool) {
	s.ready = ready
}

// Test_createProvider tests createProvider().
func Test_createProvider(t *testing.T) {
	type testCase struct {
		name         string
		config       *hetzner.Configuration
		expectedType provider.Provider
	}

	run := func(t *testing.T, tc testCase) {
		actual, _ := createProvider(tc.config)
		assert.IsType(t, tc.expectedType, actual)
	}

	testCases := []testCase{
		{
			name: "hetznerdns implementation",
			config: &hetzner.Configuration{
				UseCloudAPI: false,
			},
			expectedType: &hetznerdns.HetznerProvider{},
		},
		{
			name: "hetznercloud implementation",
			config: &hetzner.Configuration{
				UseCloudAPI: true,
			},
			expectedType: &hetznercloud.HetznerProvider{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_waitForSignal tests waitForSignal().
func Test_waitForSignal(t *testing.T) {
	name := "wait for signal test"
	actual := mockStatus{
		ready:   true,
		healthy: true,
	}
	expected := mockStatus{}
	bkpNotify := notify
	notify = func(sig chan os.Signal) {
		go func() {
			time.Sleep(time.Second)
			sig <- syscall.SIGTERM
		}()
	}

	t.Run(name, func(t *testing.T) {
		waitForSignal(&actual)
		assert.Equal(t, expected, actual)
	})

	notify = bkpNotify
}
