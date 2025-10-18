/*
 * HetznerDNS - Conversion utilities - unit tests.
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
package dns

import (
	"testing"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner/model"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/jobstoit/hetzner-dns-go/dns/schema"
	"github.com/stretchr/testify/assert"
)

var (
	// Test creation time
	testCTime, _ = time.Parse("2006-01-02 15:04:05", "2024-11-02 17:09:47")
	// Test modification time
	testMTime, _ = time.Parse("2006-01-02 15:04:05", "2025-01-02 01:10:33")
)

// Test_getDNSListOpts tests getDNSListOpts().
func Test_getDNSListOpts(t *testing.T) {
	expected := hdns.ListOpts{Page: 1, PerPage: 100}
	input := model.ListOpts{PageIdx: 1, ItemsPerPage: 100}
	actual := getDNSListOpts(input)
	assert.Equal(t, expected, actual)
}

// Test_getDNSRecordListOpts tests getDNSRecordListOpts().
func Test_getDNSRecordListOpts(t *testing.T) {
	expected := hdns.RecordListOpts{
		ListOpts: hdns.ListOpts{Page: 1, PerPage: 100},
		ZoneID:   "zoneIDAlpha",
	}
	input := model.RecordListOpts{
		ListOpts: model.ListOpts{PageIdx: 1, ItemsPerPage: 100},
		ZoneID:   "zoneIDAlpha",
	}
	actual := getDNSRecordListOpts(input)
	assert.Equal(t, expected, actual)
}

// Test_getDNSZoneListOpts tests getDNSZoneListOpts().
func Test_getDNSZoneListOpts(t *testing.T) {
	expected := hdns.ZoneListOpts{
		ListOpts:   hdns.ListOpts{Page: 1, PerPage: 100},
		Name:       "domain.test",
		SearchName: "searchdomain.test",
	}
	input := model.ZoneListOpts{
		ListOpts:   model.ListOpts{PageIdx: 1, ItemsPerPage: 100},
		Name:       "domain.test",
		SearchName: "searchdomain.test",
	}
	actual := getDNSZoneListOpts(input)
	assert.Equal(t, expected, actual)
}

// Test_getDNSZone tests getDNSZone().
func Test_getDNSZone(t *testing.T) {
	expected := hdns.Zone{
		ID:       "zoneIDDomain",
		Created:  schema.HdnsTime(testCTime),
		Modified: schema.HdnsTime(testMTime),
		Name:     "domain.test",
		Ttl:      10000,
	}
	input := model.Zone{
		ID:       "zoneIDDomain",
		Created:  testCTime,
		Modified: testMTime,
		Name:     "domain.test",
		TTL:      10000,
	}
	actual := getDNSZone(input)
	assert.Equal(t, expected, actual)
}

// Test_getZone tests getZone().
func Test_getZone(t *testing.T) {
	expected := model.Zone{
		ID:       "zoneIDDomain",
		Created:  testCTime,
		Modified: testMTime,
		Name:     "domain.test",
		TTL:      10000,
	}
	input := hdns.Zone{
		ID:       "zoneIDDomain",
		Created:  schema.HdnsTime(testCTime),
		Modified: schema.HdnsTime(testMTime),
		Name:     "domain.test",
		Ttl:      10000,
	}
	actual := getZone(input)
	assert.Equal(t, expected, actual)
}

// Test_getPZoneArray tests getPZoneArray().
func Test_getPZoneArray(t *testing.T) {
	expected := []model.Zone{
		{
			ID:       "zoneIDAlpha",
			Created:  testCTime,
			Modified: testMTime,
			Name:     "alpha.test",
			TTL:      10000,
		},
		{
			ID:       "zoneIDBeta",
			Created:  testCTime,
			Modified: testMTime,
			Name:     "beta.test",
			TTL:      7200,
		},
	}
	input := []*hdns.Zone{
		{
			ID:       "zoneIDAlpha",
			Created:  schema.HdnsTime(testCTime),
			Modified: schema.HdnsTime(testMTime),
			Name:     "alpha.test",
			Ttl:      10000,
		},
		{
			ID:       "zoneIDBeta",
			Created:  schema.HdnsTime(testCTime),
			Modified: schema.HdnsTime(testMTime),
			Name:     "beta.test",
			Ttl:      7200,
		},
	}
	actual := getPZoneArray(input)
	assert.EqualValues(t, expected, actual)
}

// Test_getDNSTtl tests getDNSTttl().
func Test_getDNSTtl(t *testing.T) {
	type testCase struct {
		name     string
		input    int
		expected *int
	}

	run := func(t *testing.T, tc testCase) {
		actual := getDNSTtl(tc.input)
		assert.EqualValues(t, tc.expected, actual)
	}

	int7200 := 7200

	testCases := []testCase{
		{
			name:     "TTL <= 0",
			input:    -1,
			expected: nil,
		},
		{
			name:     "TTL > 0",
			input:    7200,
			expected: &int7200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getRecord tests getRecord().
func Test_getRecord(t *testing.T) {
	type testCase struct {
		name     string
		input    hdns.Record
		expected model.Record
	}

	run := func(t *testing.T, tc testCase) {
		actual := getRecord(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "Record with nil zone",
			expected: model.Record{
				ID:       "id_1",
				Name:     "www",
				Created:  testCTime,
				Modified: testMTime,
				Type:     "A",
				Value:    "1.1.1.1",
				TTL:      7200,
			},
			input: hdns.Record{
				ID:       "id_1",
				Name:     "www",
				Created:  schema.HdnsTime(testCTime),
				Modified: schema.HdnsTime(testMTime),
				Type:     "A",
				Value:    "1.1.1.1",
				Ttl:      7200,
			},
		}, {
			name: "Record with non-nil zone",
			expected: model.Record{
				ID:       "id_1",
				Name:     "www",
				Created:  testCTime,
				Modified: testMTime,
				Zone: &model.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
					TTL:  10000,
				},
				Type:  "A",
				Value: "1.1.1.1",
				TTL:   7200,
			},
			input: hdns.Record{
				ID:       "id_1",
				Name:     "www",
				Created:  schema.HdnsTime(testCTime),
				Modified: schema.HdnsTime(testMTime),
				Zone: &hdns.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
					Ttl:  10000,
				},
				Type:  "A",
				Value: "1.1.1.1",
				Ttl:   7200,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getPRecordArray tests getPRecordArray().
func Test_getPRecordArray(t *testing.T) {
	expected := []model.Record{
		{
			ID:       "id_1",
			Name:     "www",
			Created:  testCTime,
			Modified: testMTime,
			Zone: &model.Zone{
				ID:   "zoneIDAlpha",
				Name: "alpha.test",
				TTL:  10000,
			},
			Type:  "A",
			Value: "1.1.1.1",
			TTL:   7200,
		},
		{
			ID:       "id_2",
			Name:     "ftp",
			Created:  testCTime,
			Modified: testMTime,
			Zone: &model.Zone{
				ID:   "zoneIDAlpha",
				Name: "alpha.test",
				TTL:  10000,
			},
			Type:  "A",
			Value: "2.2.2.2",
			TTL:   7200,
		}}
	input := []*hdns.Record{
		{
			ID:       "id_1",
			Name:     "www",
			Created:  schema.HdnsTime(testCTime),
			Modified: schema.HdnsTime(testMTime),
			Zone: &hdns.Zone{
				ID:   "zoneIDAlpha",
				Name: "alpha.test",
				Ttl:  10000,
			},
			Type:  "A",
			Value: "1.1.1.1",
			Ttl:   7200,
		},
		{
			ID:       "id_2",
			Name:     "ftp",
			Created:  schema.HdnsTime(testCTime),
			Modified: schema.HdnsTime(testMTime),
			Zone: &hdns.Zone{
				ID:   "zoneIDAlpha",
				Name: "alpha.test",
				Ttl:  10000,
			},
			Type:  "A",
			Value: "2.2.2.2",
			Ttl:   7200,
		},
	}
	actual := getPRecordArray(input)
	assert.EqualValues(t, expected, actual)
}

// Test_getDNSRecordCreateOpts tests getDNSRecordCreateOpts().
func Test_getDNSRecordCreateOpts(t *testing.T) {
	type testCase struct {
		name     string
		input    model.Record
		expected hdns.RecordCreateOpts
	}

	run := func(t *testing.T, tc testCase) {
		actual := getDNSRecordCreateOpts(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	int7200 := 7200

	testCases := []testCase{
		{
			name: "Record with nil zone",
			input: model.Record{
				Name:  "www",
				Type:  "A",
				Value: "1.1.1.1",
				TTL:   7200,
			},
			expected: hdns.RecordCreateOpts{
				Name:  "www",
				Type:  hdns.RecordType("A"),
				Value: "1.1.1.1",
				Ttl:   &int7200,
			},
		},
		{
			name: "Record with non-nil zone",
			input: model.Record{
				Name: "www",
				Zone: &model.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
				},
				Type:  "A",
				Value: "1.1.1.1",
				TTL:   7200,
			},
			expected: hdns.RecordCreateOpts{
				Name: "www",
				Zone: &hdns.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
				},
				Type:  hdns.RecordType("A"),
				Value: "1.1.1.1",
				Ttl:   &int7200,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getDNSRecordUpdateOpts tests getDNSRecordUpdateOpts().
func Test_getDNSRecordUpdateOpts(t *testing.T) {
	type testCase struct {
		name     string
		input    model.Record
		expected hdns.RecordUpdateOpts
	}

	run := func(t *testing.T, tc testCase) {
		actual := getDNSRecordUpdateOpts(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	int7200 := 7200

	testCases := []testCase{
		{
			name: "Record with nil zone",
			input: model.Record{
				ID:    "id_1",
				Name:  "www",
				Type:  "A",
				Value: "1.1.1.1",
				TTL:   7200,
			},
			expected: hdns.RecordUpdateOpts{
				Name:  "www",
				Type:  hdns.RecordType("A"),
				Value: "1.1.1.1",
				Ttl:   &int7200,
			},
		},
		{
			name: "Record with non-nil zone",
			input: model.Record{
				ID:   "id_1",
				Name: "www",
				Zone: &model.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
				},
				Type:  "A",
				Value: "1.1.1.1",
				TTL:   7200,
			},
			expected: hdns.RecordUpdateOpts{
				Name: "www",
				Zone: &hdns.Zone{
					ID:   "zoneIDAlpha",
					Name: "alpha.test",
				},
				Type:  hdns.RecordType("A"),
				Value: "1.1.1.1",
				Ttl:   &int7200,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_getPaginationMeta tests getPaginationMeta().
func Test_getPaginationMeta(t *testing.T) {
	type testCase struct {
		name     string
		input    hdns.Meta
		expected *model.Pagination
	}

	run := func(t *testing.T, tc testCase) {
		actual := getPaginationMeta(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "No pagination info",
			input:    hdns.Meta{},
			expected: nil,
		},
		{
			name: "Pagination info present",
			input: hdns.Meta{
				Pagination: &hdns.Pagination{
					Page:         1,
					PerPage:      100,
					LastPage:     2,
					TotalEntries: 120,
				},
			},
			expected: &model.Pagination{
				ItemsPerPage: 100,
				PageIdx:      1,
				LastPage:     2,
				TotalCount:   120,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
