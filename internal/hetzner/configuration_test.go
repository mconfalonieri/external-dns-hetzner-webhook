/*
 * Configuration - unit tests
 *
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
package hetzner

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
)

// Test_GetDomainFilter tests that the domain filter is correctly set for all
// cases.
func Test_GetDomainFilter(t *testing.T) {
	t.Setenv("HETZNER_API_KEY", "test-key")
	type testCase struct {
		name     string
		config   Configuration
		expected *endpoint.DomainFilter
	}

	run := func(t *testing.T, tc testCase) {
		actual := GetDomainFilter(tc.config)
		actualJSON, _ := actual.MarshalJSON()
		expectedJSON, _ := tc.expected.MarshalJSON()
		assert.Equal(t, actualJSON, expectedJSON)
	}

	testCases := []testCase{
		{
			name:     "No domain filters",
			config:   Configuration{},
			expected: &endpoint.DomainFilter{},
		},
		{
			name: "Simple domain filter",
			config: Configuration{
				DomainFilter: []string{"example.com"},
			},
			expected: endpoint.NewDomainFilter([]string{"example.com"}),
		},
		{
			name: "Exclusion domain filter",
			config: Configuration{
				ExcludeDomains: []string{"example.com"},
			},
			expected: endpoint.NewDomainFilterWithExclusions(nil, []string{"example.com"}),
		},
		{
			name: "Both domain filters",
			config: Configuration{
				DomainFilter:   []string{"example-included.com"},
				ExcludeDomains: []string{"example-excluded.com"},
			},
			expected: endpoint.NewDomainFilterWithExclusions(
				[]string{"example-included.com"},
				[]string{"example-excluded.com"},
			),
		},
		{
			name: "Regular expression domain filters",
			config: Configuration{
				RegexDomainFilter:    `example-[a-z]+\.com`,
				RegexDomainExclusion: `[a-z]+-excluded\.com`,
			},
			expected: endpoint.NewRegexDomainFilter(
				regexp.MustCompile(`example-[a-z]+\.com`),
				regexp.MustCompile(`[a-z]+-excluded\.com`),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
