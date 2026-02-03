/*
 * Labels - unit tests.
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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
)

// Test_formatLabels tests formatLabels().
func Test_formatLabels(t *testing.T) {
	type testCase struct {
		name     string
		input    map[string]string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := formatLabels(tc.input)
		assert.ElementsMatch(t, strings.Split(tc.expected, ";"), strings.Split(actual, ";"))
	}

	testCases := []testCase{
		{
			name:     "nil map",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: "",
		},
		{
			name:     "map with one element",
			input:    map[string]string{"label": "value"},
			expected: "label=value",
		},
		{
			name: "map with multiple elements",
			input: map[string]string{
				"label1": "value1",
				"label2": "value2",
				"label3": "value3",
			},
			expected: "label1=value1;label2=value2;label3=value3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getProviderSpecific tests getProviderSpecific().
func Test_getProviderSpecific(t *testing.T) {
	type testCase struct {
		name     string
		input    map[string]string
		expected endpoint.ProviderSpecific
	}

	run := func(t *testing.T, tc testCase) {
		actual := getProviderSpecific("--slash--", tc.input)
		assert.ElementsMatch(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: nil,
		},
		{
			name: "simple elements",
			input: map[string]string{
				"env":     "test",
				"project": "vanilla",
			},
			expected: endpoint.ProviderSpecific{
				endpoint.ProviderSpecificProperty{
					Name:  "webhook/hetzner-label-env",
					Value: "test",
				},
				endpoint.ProviderSpecificProperty{
					Name:  "webhook/hetzner-label-project",
					Value: "vanilla",
				},
			},
		},
		{
			name: "complex elements",
			input: map[string]string{
				"alpha.com/app-env":  "test",
				"project/subproject": "ice-cream.vanilla",
			},
			expected: endpoint.ProviderSpecific{
				endpoint.ProviderSpecificProperty{
					Name:  "webhook/hetzner-label-alpha.com--slash--app-env",
					Value: "test",
				},
				endpoint.ProviderSpecificProperty{
					Name:  "webhook/hetzner-label-project--slash--subproject",
					Value: "ice-cream.vanilla",
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

// Test_checkLabel tests checkLabel().
func Test_checkLabel(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected error
	}

	run := func(t *testing.T, tc testCase) {
		actual := checkLabel(tc.input)
		assertError(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "label not acceptable 1",
			input:    "@",
			expected: errors.New("label [@] is not acceptable"),
		},
		{
			name:     "label not acceptable 2",
			input:    "@label",
			expected: errors.New("label [@label] is not acceptable"),
		},
		{
			name:     "label not acceptable 3",
			input:    "label@",
			expected: errors.New("label [label@] is not acceptable"),
		},
		{
			name:     "label not acceptable 4",
			input:    ".label",
			expected: errors.New("label [.label] is not acceptable"),
		},
		{
			name:     "label not acceptable 5",
			input:    "label/",
			expected: errors.New("label [label/] is not acceptable"),
		},
		{
			name:     "label not acceptable 6",
			input:    "this-is-a-very-long-label-label-label-label-label-label-label-label",
			expected: errors.New("label [this-is-a-very-long-...] is longer than 63 characters"),
		},
		{
			name:     "acceptable label 1",
			input:    "l",
			expected: nil,
		},
		{
			name:     "acceptable label 2",
			input:    "label",
			expected: nil,
		},
		{
			name:     "acceptable label 3",
			input:    "prefix/label",
			expected: nil,
		},
		{
			name:     "acceptable label 4",
			input:    "prefix.label",
			expected: nil,
		},
		{
			name:     "acceptable label 5",
			input:    "prefix_label",
			expected: nil,
		},
		{
			name:     "acceptable label 6",
			input:    "prefix-label",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_checkValue(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected error
	}

	run := func(t *testing.T, tc testCase) {
		actual := checkValue(tc.input)
		assertError(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "value not acceptable 1",
			input:    "@",
			expected: errors.New("value \"@\" is not acceptable"),
		},
		{
			name:     "value not acceptable 2",
			input:    "@value",
			expected: errors.New("value \"@value\" is not acceptable"),
		},
		{
			name:     "value not acceptable 3",
			input:    "-value",
			expected: errors.New("value \"-value\" is not acceptable"),
		},
		{
			name:     "value not acceptable 4",
			input:    "prefix/value",
			expected: errors.New("value \"prefix/value\" is not acceptable"),
		},
		{
			name:     "value not acceptable 5",
			input:    "this-is-a-very-long-value-value-value-value-value-value-value-value",
			expected: errors.New("value \"this-is-a-very-long-...\" is longer than 63 characters"),
		},
		{
			name:     "acceptable value 1",
			input:    "",
			expected: nil,
		},
		{
			name:     "acceptable value 2",
			input:    "value",
			expected: nil,
		},
		{
			name:     "acceptable value 3",
			input:    "v",
			expected: nil,
		},
		{
			name:     "acceptable value 4",
			input:    "prefix.value",
			expected: nil,
		},
		{
			name:     "acceptable value 5",
			input:    "prefix_value",
			expected: nil,
		},
		{
			name:     "acceptable value 6",
			input:    "prefix-value",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_extractHetznerLabels tests extractHetznerLabels().
func Test_extractHetznerLabels(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			slash string
			ps    endpoint.ProviderSpecific
		}
		expected struct {
			labels map[string]string
			err    error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		inp := tc.input
		labels, err := extractHetznerLabels(inp.slash, inp.ps)
		if !assertError(t, exp.err, err) {
			assert.Equal(t, exp.labels, labels)
		}
	}

	testCases := []testCase{
		{
			name: "no provider specific parameters",
			input: struct {
				slash string
				ps    endpoint.ProviderSpecific
			}{
				slash: "--slash--",
				ps:    endpoint.ProviderSpecific{},
			},
			expected: struct {
				labels map[string]string
				err    error
			}{
				labels: map[string]string{},
			},
		},
		{
			name: "unsupported provider specific parameters",
			input: struct {
				slash string
				ps    endpoint.ProviderSpecific
			}{
				slash: "--slash--",
				ps: endpoint.ProviderSpecific{
					endpoint.ProviderSpecificProperty{
						Name:  "test/custom-parameter",
						Value: "value",
					},
				},
			},
			expected: struct {
				labels map[string]string
				err    error
			}{
				labels: map[string]string{},
			},
		},
		{
			name: "mixed provider specific parameters",
			input: struct {
				slash string
				ps    endpoint.ProviderSpecific
			}{
				slash: "--testslash--",
				ps: endpoint.ProviderSpecific{
					endpoint.ProviderSpecificProperty{
						Name:  "test/custom-parameter",
						Value: "value",
					},
					endpoint.ProviderSpecificProperty{
						Name:  "webhook/hetzner-label-environment",
						Value: "test",
					},
					endpoint.ProviderSpecificProperty{
						Name:  "webhook/hetzner-label-project--testslash--subproject",
						Value: "prefix.value",
					},
				},
			},
			expected: struct {
				labels map[string]string
				err    error
			}{
				labels: map[string]string{
					"project/subproject": "prefix.value",
					"environment":        "test",
				},
			},
		},
		{
			name: "empty slash parameter",
			input: struct {
				slash string
				ps    endpoint.ProviderSpecific
			}{
				slash: "",
				ps: endpoint.ProviderSpecific{
					endpoint.ProviderSpecificProperty{
						Name:  "webhook/hetzner-label-project--slash--subproject",
						Value: "prefix.value",
					},
				},
			},
			expected: struct {
				labels map[string]string
				err    error
			}{
				labels: map[string]string{
					"project/subproject": "prefix.value",
				},
			},
		},
		{
			name: "value format error",
			input: struct {
				slash string
				ps    endpoint.ProviderSpecific
			}{
				slash: "--slash--",
				ps: endpoint.ProviderSpecific{
					endpoint.ProviderSpecificProperty{
						Name:  "webhook/hetzner-label-project",
						Value: "prefix/value",
					},
				},
			},
			expected: struct {
				labels map[string]string
				err    error
			}{
				err: errors.New("cannot process value for [project: \"prefix/value\"]: value \"prefix/value\" is not acceptable"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
