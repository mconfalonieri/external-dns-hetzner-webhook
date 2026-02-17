/*
 * Rate limit - Unit tests
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
package metrics

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_readLimit tests readLimit().
func Test_readLimit(t *testing.T) {
	type testCase struct {
		name     string
		input    http.Header
		expected struct {
			limit int
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		limit, err := readLimit(tc.input)
		assert.Equal(t, exp.limit, limit)
		assert.Equal(t, exp.err, err)
	}

	testCases := []testCase{
		{
			name: "limit ok",
			input: http.Header{
				"Ratelimit-Limit": {"1000"},
			},
			expected: struct {
				limit int
				err   error
			}{
				limit: 1000,
				err:   nil,
			},
		},
		{
			name:  "limit not found",
			input: http.Header{},
			expected: struct {
				limit int
				err   error
			}{
				limit: 0,
				err:   errors.New("header Ratelimit-Limit not found"),
			},
		},
		{
			name: "unexpected value",
			input: http.Header{
				"Ratelimit-Limit": {"TXT"},
			},
			expected: struct {
				limit int
				err   error
			}{
				limit: 0,
				err:   errors.New("header Ratelimit-Limit had unexpected value \"TXT\""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_readRemaining tests readRemaining().
func Test_readRemaining(t *testing.T) {
	type testCase struct {
		name     string
		input    http.Header
		expected struct {
			remaining int
			err       error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		remaining, err := readRemaining(tc.input)
		assert.Equal(t, exp.remaining, remaining)
		assert.Equal(t, exp.err, err)
	}

	testCases := []testCase{
		{
			name: "remaining ok",
			input: http.Header{
				"Ratelimit-Remaining": {"500"},
			},
			expected: struct {
				remaining int
				err       error
			}{
				remaining: 500,
				err:       nil,
			},
		},
		{
			name:  "remaining not found",
			input: http.Header{},
			expected: struct {
				remaining int
				err       error
			}{
				remaining: 0,
				err:       errors.New("header Ratelimit-Remaining not found"),
			},
		},
		{
			name: "unexpected value",
			input: http.Header{
				"Ratelimit-Remaining": {"TXT"},
			},
			expected: struct {
				remaining int
				err       error
			}{
				remaining: 0,
				err:       errors.New("header Ratelimit-Remaining had unexpected value \"TXT\""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_readReset tests readReset().
func Test_readReset(t *testing.T) {
	type testCase struct {
		name     string
		input    http.Header
		expected struct {
			reset uint64
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		reset, err := readReset(tc.input)
		assert.Equal(t, exp.reset, reset)
		assert.Equal(t, exp.err, err)
	}

	testCases := []testCase{
		{
			name: "reset ok",
			input: http.Header{
				"Ratelimit-Reset": {"1771370227"},
			},
			expected: struct {
				reset uint64
				err   error
			}{
				reset: 1771370227,
				err:   nil,
			},
		},
		{
			name:  "reset not found",
			input: http.Header{},
			expected: struct {
				reset uint64
				err   error
			}{
				reset: 0,
				err:   errors.New("header Ratelimit-Reset not found"),
			},
		},
		{
			name: "unexpected value",
			input: http.Header{
				"Ratelimit-Reset": {"TXT"},
			},
			expected: struct {
				reset uint64
				err   error
			}{
				reset: 0,
				err:   errors.New("header Ratelimit-Reset had unexpected value \"TXT\""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_readReset tests readReset().
func Test_parseRateLimit(t *testing.T) {
	type testCase struct {
		name     string
		input    http.Header
		expected struct {
			rl  *rateLimit
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		rl, err := parseRateLimit(tc.input)
		assert.Equal(t, exp.rl, rl)
		assert.Equal(t, exp.err, err)
	}

	testCases := []testCase{
		{
			name: "parse ok",
			input: http.Header{
				"Ratelimit-Limit":     {"1000"},
				"Ratelimit-Remaining": {"500"},
				"Ratelimit-Reset":     {"1771370227"},
			},
			expected: struct {
				rl  *rateLimit
				err error
			}{
				rl: &rateLimit{
					limit:     1000,
					remaining: 500,
					reset:     uint64(1771370227),
				},
				err: nil,
			},
		},
		{
			name: "limit error",
			input: http.Header{
				"Ratelimit-Remaining": {"500"},
				"Ratelimit-Reset":     {"1771370227"},
			},
			expected: struct {
				rl  *rateLimit
				err error
			}{
				rl:  nil,
				err: errors.New("header Ratelimit-Limit not found"),
			},
		},
		{
			name: "remaining error",
			input: http.Header{
				"Ratelimit-Limit": {"1000"},
				"Ratelimit-Reset": {"1771370227"},
			},
			expected: struct {
				rl  *rateLimit
				err error
			}{
				rl:  nil,
				err: errors.New("header Ratelimit-Remaining not found"),
			},
		},
		{
			name: "reset error",
			input: http.Header{
				"Ratelimit-Limit":     {"1000"},
				"Ratelimit-Remaining": {"500"},
			},
			expected: struct {
				rl  *rateLimit
				err error
			}{
				rl:  nil,
				err: errors.New("header Ratelimit-Reset not found"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
