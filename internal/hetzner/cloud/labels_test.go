/*
 * Labels - unit tests.
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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, tc.expected, actual)
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

// Test_parsePair tests parsePair().
func Test_parsePair(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected struct {
			label string
			value string
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		label, value, err := parsePair(tc.input)
		if !assertError(t, exp.err, err) {
			assert.Equal(t, exp.label, label)
			assert.Equal(t, exp.value, value)
		}
	}

	testCases := []testCase{
		{
			name:  "empty string",
			input: "",
			expected: struct {
				label string
				value string
				err   error
			}{
				err: errors.New("empty string provided"),
			},
		},
		{
			name:  "malformed pair 1",
			input: "label",
			expected: struct {
				label string
				value string
				err   error
			}{
				err: errors.New("malformed pair \"label\""),
			},
		},
		{
			name:  "malformed pair 2",
			input: "label=value=value",
			expected: struct {
				label string
				value string
				err   error
			}{
				err: errors.New("malformed pair \"label=value=value\""),
			},
		},
		{
			name:  "label error",
			input: "=value",
			expected: struct {
				label string
				value string
				err   error
			}{
				err: errors.New("in pair \"=value\": empty label is not acceptable"),
			},
		},
		{
			name:  "value error",
			input: "label=prefix/value",
			expected: struct {
				label string
				value string
				err   error
			}{
				err: errors.New("for label [label]: value \"prefix/value\" is not acceptable"),
			},
		},
		{
			name:  "acceptable pair",
			input: "prefix/label=value",
			expected: struct {
				label string
				value string
				err   error
			}{
				label: "prefix/label",
				value: "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
