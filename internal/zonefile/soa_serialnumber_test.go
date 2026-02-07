/*
 * SOASerialNumber - Test suite.
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
package zonefile

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_collectDate(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected struct {
			datePart string
			err      error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		datePart, err := collectDate(tc.input)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.datePart, datePart)
	}

	testCases := []testCase{
		{
			name:  "invalid serial number string",
			input: "AAAAAAAAAA",
			expected: struct {
				datePart string
				err      error
			}{
				datePart: "",
				err:      errors.New("cannot parse date in serial number \"AAAAAAAAAA\""),
			},
		},
		{
			name:  "future serial number",
			input: "2100121155",
			expected: struct {
				datePart string
				err      error
			}{
				datePart: "",
				err:      errors.New("unexpected date part \"21001211\" is in the future"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_NewSOASerialNumber(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected struct {
			soaSerialNumber *SOASerialNumber
			err             error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		soaSerialNumber, err := NewSOASerialNumber(tc.input)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.soaSerialNumber, soaSerialNumber)
	}

	testCases := []testCase{
		{
			name:  "empty string",
			input: "",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: nil,
				err:             errors.New("serial number \"\" is unsupported"),
			},
		},
		{
			name:  "invalid date",
			input: "AAAAAAAAAA",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: nil,
				err:             errors.New("cannot parse date in serial number \"AAAAAAAAAA\""),
			},
		},
		{
			name:  "unsupported version",
			input: "20260118-1",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: nil,
				err:             errors.New("version -1 is not supported"),
			},
		},
		{
			name:  "valid serial number version zero",
			input: "2026011800",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: &SOASerialNumber{
					date:    "20260118",
					version: 0,
				},
				err: nil,
			},
		},
		{
			name:  "valid serial number version single digit",
			input: "2026011805",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: &SOASerialNumber{
					date:    "20260118",
					version: 5,
				},
				err: nil,
			},
		},
		{
			name:  "valid serial number version double digit",
			input: "2026011845",
			expected: struct {
				soaSerialNumber *SOASerialNumber
				err             error
			}{
				soaSerialNumber: &SOASerialNumber{
					date:    "20260118",
					version: 45,
				},
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_CreateSOASerialNumber(t *testing.T) {
	type testCase struct {
		name     string
		expected *SOASerialNumber
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		actual := CreateSOASerialNumber()
		assert.Equal(t, exp, actual)
	}

	testCases := []testCase{
		{
			name: "create",
			expected: &SOASerialNumber{
				date:    time.Now().Format(fmtSOADate),
				version: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_SOASerialNumber_Inc(t *testing.T) {
	type testCase struct {
		name     string
		object   SOASerialNumber
		expected struct {
			err error
			obj SOASerialNumber
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		obj := tc.object
		actual := obj.Inc()
		if !assertError(t, exp.err, actual) {
			assert.Equal(t, exp.obj, obj)
		}
	}

	testCases := []testCase{
		{
			name: "update from past date",
			object: SOASerialNumber{
				date:    "20201201",
				version: 50,
			},
			expected: struct {
				err error
				obj SOASerialNumber
			}{
				obj: SOASerialNumber{
					date:    time.Now().Format(fmtSOADate),
					version: 0,
				},
			},
		},
		{
			name: "forbidden update",
			object: SOASerialNumber{
				date:    time.Now().Format(fmtSOADate),
				version: 99,
			},
			expected: struct {
				err error
				obj SOASerialNumber
			}{
				err: errors.New("cannot increment version as it is 99"),
				obj: SOASerialNumber{
					date:    time.Now().Format(fmtSOADate),
					version: 99,
				},
			},
		},
		{
			name: "same date update",
			object: SOASerialNumber{
				date:    time.Now().Format(fmtSOADate),
				version: 45,
			},
			expected: struct {
				err error
				obj SOASerialNumber
			}{
				obj: SOASerialNumber{
					date:    time.Now().Format(fmtSOADate),
					version: 46,
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

func Test_SOASerialNumber_String(t *testing.T) {
	type testCase struct {
		name     string
		object   SOASerialNumber
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		obj := tc.object
		actual := obj.String()
		assert.Equal(t, exp, actual)
	}

	testCases := []testCase{
		{
			name: "string conversion double digit",
			object: SOASerialNumber{
				date:    "20201201",
				version: 50,
			},
			expected: "2020120150",
		},
		{
			name: "string conversion single digit",
			object: SOASerialNumber{
				date:    "20201201",
				version: 5,
			},
			expected: "2020120105",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_SOASerialNumber_Uint32(t *testing.T) {
	type testCase struct {
		name     string
		object   SOASerialNumber
		expected uint32
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		obj := tc.object
		actual := obj.Uint32()
		assert.Equal(t, exp, actual)
	}

	testCases := []testCase{
		{
			name: "uint32 conversion double digit",
			object: SOASerialNumber{
				date:    "20201201",
				version: 50,
			},
			expected: uint32(2020120150),
		},
		{
			name: "uint32 conversion single digit",
			object: SOASerialNumber{
				date:    "20201201",
				version: 5,
			},
			expected: uint32(2020120105),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
