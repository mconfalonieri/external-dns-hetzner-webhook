/*
 * Bulk changes - unit tests.
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
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"external-dns-hetzner-webhook/internal/zonefile"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

const (
	fmtSOADate    = "20060102"
	oldSOA        = "2025112009"
	inputZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
@	3600	IN	A	116.202.181.2
www	3600	IN	A	116.202.181.2
`
	inputZoneFileNoTTL = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
@	3600	IN	A	116.202.181.2
www	3600	IN	A	116.202.181.2
`
	createZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
@	3600	IN	A	116.202.181.2
www	3600	IN	A	116.202.181.2
ftp 7200    IN  A   116.202.181.3
`
	updatedRecordsetZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
@	3600	IN	A	116.202.181.2
www	3600	IN	A	116.202.181.2
www	3600	IN	A	116.202.181.3
`
	updatedTTLZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
@	3600	IN	A	116.202.181.2
www	7200	IN	A	116.202.181.2
`
	deletedZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

@	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
@	3600	IN	NS	helium.ns.hetzner.de.
@	3600	IN	NS	hydrogen.ns.hetzner.com.
@	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
@	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
www	3600	IN	A	116.202.181.2
`

	changedZoneFile = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	86400

fastipletonis.eu.	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
fastipletonis.eu.	3600	IN	NS	helium.ns.hetzner.de.
fastipletonis.eu.	3600	IN	NS	hydrogen.ns.hetzner.com.
fastipletonis.eu.	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
fastipletonis.eu.	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
ftp.fastipletonis.eu.	3600	IN	A	116.202.181.1
www.fastipletonis.eu.	3600	IN	A	116.202.181.2
www.fastipletonis.eu.	3600	IN	A	116.202.181.3
`
	changedZoneFileDefaultTTL = `;; Exported on 2026-01-19T21:39:41Z
$ORIGIN	fastipletonis.eu.
$TTL	7200

fastipletonis.eu.	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600

; NS records
fastipletonis.eu.	3600	IN	NS	helium.ns.hetzner.de.
fastipletonis.eu.	3600	IN	NS	hydrogen.ns.hetzner.com.
fastipletonis.eu.	3600	IN	NS	oxygen.ns.hetzner.com.

; CAA records
fastipletonis.eu.	3600	IN	CAA	128 issue "letsencrypt.org"

; A records
ftp.fastipletonis.eu.	3600	IN	A	116.202.181.1
www.fastipletonis.eu.	3600	IN	A	116.202.181.2
www.fastipletonis.eu.	3600	IN	A	116.202.181.3
`
)

var (
	ttl3600 = 3600
	ttl7200 = 7200
)

// sortRows sorts the file rows for comparison.
func sortRows(file string) []string {
	array := strings.Split(file, "\n")
	slices.Sort(array)
	// Remove comments and empty rows
	reduced := make([]string, 0)
	for _, str := range array {
		if str != "" && !strings.HasPrefix(str, ";") {
			reduced = append(reduced, str)
		}
	}
	return reduced
}

// todayMinSerialNumber returns today's minimum serial number as string.
func todayMinSerialNumber() string {
	return time.Now().Format(fmtSOADate) + "00"
}

// todayMaxSerialNumber returns today's maximum serial number as string.
func todayMaxSerialNumber() string {
	return time.Now().Format(fmtSOADate) + "99"
}

// createTestZonefile creates a test Zonefile.
func createTestZonefile(zfile string) *zonefile.Zonefile {
	r := strings.NewReader(zfile)
	z, _ := zonefile.NewZonefile(r, "fastipletonis.eu", 86400)
	return z
}

// Test_bulkChanges_empty tests bulkChanges.empty().
func Test_bulkChanges_empty(t *testing.T) {
	type testCase struct {
		name     string
		object   bulkChanges
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.empty()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty",
			object:   bulkChanges{},
			expected: true,
		},
		{
			name: "not empty",
			object: bulkChanges{
				changes: map[int64]*zoneChanges{
					1: {},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_bulkChanges_getZoneChanges tests bulkChanges.getZoneChanges().
func Test_bulkChanges_getZoneChanges(t *testing.T) {
	type testCase struct {
		name      string
		object    bulkChanges
		input     *hcloud.Zone
		expected  *zoneChanges
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.getZoneChanges(tc.input)
		assert.Equal(t, tc.expected, actual)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "not present",
			object: bulkChanges{
				zones:   map[int64]*hcloud.Zone{},
				changes: map[int64]*zoneChanges{},
			},
			input: &hcloud.Zone{ID: 1, Name: "alpha.com"},
			expected: &zoneChanges{
				creates: []*hetznerChangeCreate{},
				updates: []*hetznerChangeUpdate{},
				deletes: []*hetznerChangeDelete{},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
		},
		{
			name: "present",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: &hcloud.Zone{ID: 1, Name: "alpha.com"},
			expected: &zoneChanges{
				creates: []*hetznerChangeCreate{
					{
						zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
						opts: hcloud.ZoneRRSetCreateOpts{
							Name: "www",
							Type: hcloud.ZoneRRSetTypeA,
							Records: []hcloud.ZoneRRSetRecord{
								{
									Value: "10.0.0.1",
								},
							},
							TTL: &ttl3600,
						},
					},
				},
				updates: []*hetznerChangeUpdate{},
				deletes: []*hetznerChangeDelete{},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
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

// Test_bulkChanges_AddChangeCreate tests bulkChanges.AddChangeCreate().
func Test_bulkChanges_AddChangeCreate(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			opts hcloud.ZoneRRSetCreateOpts
		}
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.AddChangeCreate(inp.zone, inp.opts)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add create",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				opts hcloud.ZoneRRSetCreateOpts
			}{
				zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
				opts: hcloud.ZoneRRSetCreateOpts{
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
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

// Test_bulkChanges_AddChangeUpdate tests bulkChanges.AddChangeUpdate().
func Test_bulkChanges_AddChangeUpdate(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			rrset       *hcloud.ZoneRRSet
			ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
			recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
			updateOpts  *hcloud.ZoneRRSetUpdateOpts
		}
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.AddChangeUpdate(inp.rrset, inp.ttlOpts, inp.recordsOpts, inp.updateOpts)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add ttl update",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				rrset       *hcloud.ZoneRRSet
				ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
				recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
				updateOpts  *hcloud.ZoneRRSetUpdateOpts
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
				ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &ttl7200,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
		},
		{
			name: "add ttl update",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				rrset       *hcloud.ZoneRRSet
				ttlOpts     *hcloud.ZoneRRSetChangeTTLOpts
				recordsOpts *hcloud.ZoneRRSetSetRecordsOpts
				updateOpts  *hcloud.ZoneRRSetUpdateOpts
			}{
				rrset: &hcloud.ZoneRRSet{
					Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
					Name: "www",
					Type: hcloud.ZoneRRSetTypeA,
					Records: []hcloud.ZoneRRSetRecord{
						{
							Value: "10.0.0.1",
						},
					},
					TTL: &ttl3600,
				},
				ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &ttl7200,
				},
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
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

// Test_bulkChanges_AddChangeDelete tests bulkChanges.AddChangeDelete().
func Test_bulkChanges_AddChangeDelete(t *testing.T) {
	type testCase struct {
		name      string
		object    bulkChanges
		input     *hcloud.ZoneRRSet
		expObject bulkChanges
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		obj.AddChangeDelete(tc.input)
		assert.Equal(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add delete",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: &hcloud.ZoneRRSet{
				Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
				Name: "www",
				Type: hcloud.ZoneRRSetTypeA,
				Records: []hcloud.ZoneRRSetRecord{
					{
						Value: "10.0.0.1",
					},
				},
				TTL: &ttl3600,
			},
			expObject: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
					},
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

// Test_readTTL tests readTTL().
func Test_readTTL(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected struct {
			ttl     int
			present bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		ttl, present := readTTL(tc.input)
		assert.Equal(t, exp.ttl, ttl)
		assert.Equal(t, exp.present, present)
	}

	testCases := []testCase{
		{
			name:  "ttl present and valid",
			input: "$TTL 3600",
			expected: struct {
				ttl     int
				present bool
			}{
				ttl:     3600,
				present: true,
			},
		},
		{
			name:  "ttl not parseable",
			input: "$TTL nil",
			expected: struct {
				ttl     int
				present bool
			}{
				ttl:     0,
				present: false,
			},
		},
		{
			name:  "ttl not valid",
			input: "$TTL -3600",
			expected: struct {
				ttl     int
				present bool
			}{
				ttl:     0,
				present: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_decodeRecords tests decodeRecords().
func Test_decodeRecords(t *testing.T) {
	type testCase struct {
		name     string
		input    []hcloud.ZoneRRSetRecord
		expected []string
	}

	run := func(t *testing.T, tc testCase) {
		actual := decodeRecords(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name:     "empty set",
			input:    []hcloud.ZoneRRSetRecord{},
			expected: []string{},
		},
		{
			name: "some values",
			input: []hcloud.ZoneRRSetRecord{
				{
					Value:   "10.0.0.1",
					Comment: "Primary IP",
				},
				{
					Value:   "10.0.0.2",
					Comment: "Secondary IP",
				},
			},
			expected: []string{
				"10.0.0.1",
				"10.0.0.2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_bulkChanges_runZoneCreates tests bulkChanges.runZoneCreates().
func Test_bulkChanges_runZoneCreates(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			z    *zonefile.Zonefile
		}
		expZonefile *zonefile.Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.runZoneCreates(inp.zone, inp.z)
		assert.Equal(t, tc.expZonefile, inp.z)
	}

	testCases := []testCase{
		{
			name: "zone not found",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
									TTL: &ttl3600,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   2,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "no creates in zone",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "record created",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.3",
										},
									},
									TTL: &ttl7200,
								},
							},
						},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(createZoneFile),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_bulkChanges_runZoneUpdates tests bulkChanges.runZoneUpdates().
func Test_bulkChanges_runZoneUpdates(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			z    *zonefile.Zonefile
		}
		expZonefile *zonefile.Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.runZoneUpdates(inp.zone, inp.z)
		assert.Equal(t, tc.expZonefile, inp.z)
	}

	testCases := []testCase{
		{
			name: "zone not found",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.2",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   2,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "no updates in zone",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "recordset updated",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(updatedRecordsetZoneFile),
		},
		{
			name: "ttl updated",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								ttlOpts: &hcloud.ZoneRRSetChangeTTLOpts{
									TTL: &ttl7200,
								},
							},
						},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},

			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(updatedTTLZoneFile),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// Test_bulkChanges_runZoneDeletes tests bulkChanges.runZoneDeletes().
func Test_bulkChanges_runZoneDeletes(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			z    *zonefile.Zonefile
		}
		expZonefile *zonefile.Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		obj.runZoneDeletes(inp.zone, inp.z)
		assert.Equal(t, tc.expZonefile, inp.z)
	}

	testCases := []testCase{
		{
			name: "zone not found",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "alpha.com"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "alpha.com"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   2,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "no deletes in zone",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(inputZoneFile),
		},
		{
			name: "recordset deleted",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				z    *zonefile.Zonefile
			}{
				zone: &hcloud.Zone{
					ID:   1,
					Name: "fastipletonis.eu",
				},
				z: createTestZonefile(inputZoneFile),
			},
			expZonefile: createTestZonefile(deletedZoneFile),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_bulkChanges_runZoneChanges(t *testing.T) {
	type testCase struct {
		name   string
		object bulkChanges
		input  struct {
			zone *hcloud.Zone
			zf   string
		}
		expected struct {
			nzf string
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		nzf, err := obj.runZoneChanges(inp.zone, inp.zf)
		assertError(t, exp.err, err)
		assert.Equal(t, sortRows(exp.nzf), sortRows(nzf))
	}

	testCases := []testCase{
		{
			name: "error empty zonefile",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{},
						updates: []*hetznerChangeUpdate{},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "1",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "10.0.0.1",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				zf   string
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				zf:   "",
			},
			expected: struct {
				nzf string
				err error
			}{
				nzf: "",
				err: errors.New("cannot import zone fastipletonis.eu: cannot read records"),
			},
		},
		{
			name: "error export zonefile",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				zf   string
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				zf:   strings.Replace(inputZoneFile, oldSOA, todayMaxSerialNumber(), 1),
			},
			expected: struct {
				nzf string
				err error
			}{
				nzf: "",
				err: errors.New("cannot export zonefile: cannot increment version as it is 99"),
			},
		},
		{
			name: "regular zonefile",
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				zf   string
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				zf:   inputZoneFile,
			},
			expected: struct {
				nzf string
				err error
			}{
				nzf: strings.Replace(changedZoneFile, oldSOA, todayMinSerialNumber(), 1),
				err: nil,
			},
		},
		{
			name: "zonefile without ttl",
			object: bulkChanges{
				defaultTTL: 7200,
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				zone *hcloud.Zone
				zf   string
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				zf:   inputZoneFileNoTTL,
			},
			expected: struct {
				nzf string
				err error
			}{
				nzf: strings.Replace(changedZoneFileDefaultTTL, oldSOA, todayMinSerialNumber(), 1),
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

func Test_bulkChanges_applyZoneChanges(t *testing.T) {
	type testCase struct {
		name      string
		inpClient mockClient
		object    bulkChanges
		input     struct {
			ctx  context.Context
			zone *hcloud.Zone
		}
		expState mockClientState
		expArgs  struct {
			zone *hcloud.Zone
			opts hcloud.ZoneImportZonefileOpts
		}
	}

	run := func(t *testing.T, tc testCase) {
		client := tc.inpClient
		expState := tc.expState
		expArgs := tc.expArgs
		obj := tc.object
		inp := tc.input
		obj.dnsClient = &client
		obj.applyZoneChanges(inp.ctx, inp.zone)
		assert.Equal(t, expState, client.state)
		assert.Equal(t, expArgs.zone, client.args.ImportZoneFile.zone)
		assert.Equal(t, sortRows(expArgs.opts.Zonefile), sortRows(client.args.ImportZoneFile.opts.Zonefile))
	}

	testCases := []testCase{
		{
			name: "changes applied",
			inpClient: mockClient{
				exportZonefile: exportZonefileResponse{
					result: hcloud.ZoneExportZonefileResult{
						Zonefile: inputZoneFile,
					},
					resp: &hcloud.Response{
						Response: &http.Response{
							Status:     http.StatusText(http.StatusOK),
							StatusCode: http.StatusOK,
						},
					},
				},
			},
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				ctx  context.Context
				zone *hcloud.Zone
			}{
				ctx:  context.Background(),
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
			},
			expState: mockClientState{
				ExportZonefileCalled: true,
				ImportZonefileCalled: true,
			},
			expArgs: struct {
				zone *hcloud.Zone
				opts hcloud.ZoneImportZonefileOpts
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				opts: hcloud.ZoneImportZonefileOpts{
					Zonefile: strings.Replace(changedZoneFile, oldSOA, todayMinSerialNumber(), 1),
				},
			},
		},
		{
			name: "error export zonefile",
			inpClient: mockClient{
				exportZonefile: exportZonefileResponse{
					result: hcloud.ZoneExportZonefileResult{},
					resp: &hcloud.Response{
						Response: &http.Response{
							Status:     http.StatusText(http.StatusInternalServerError),
							StatusCode: http.StatusInternalServerError,
						},
					},
					err: errors.New("internal server error"),
				},
			},
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				ctx  context.Context
				zone *hcloud.Zone
			}{
				ctx:  context.Background(),
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
			},
			expState: mockClientState{
				ExportZonefileCalled: true,
			},
		},
		{
			name: "error run zone changes",
			inpClient: mockClient{
				exportZonefile: exportZonefileResponse{
					result: hcloud.ZoneExportZonefileResult{
						Zonefile: "",
					},
					resp: &hcloud.Response{
						Response: &http.Response{
							Status:     http.StatusText(http.StatusOK),
							StatusCode: http.StatusOK,
						},
					},
				},
			},
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				ctx  context.Context
				zone *hcloud.Zone
			}{
				ctx:  context.Background(),
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
			},
			expState: mockClientState{
				ExportZonefileCalled: true,
			},
		},
		{
			name: "error import zonefile",
			inpClient: mockClient{
				exportZonefile: exportZonefileResponse{
					result: hcloud.ZoneExportZonefileResult{
						Zonefile: inputZoneFile,
					},
					resp: &hcloud.Response{
						Response: &http.Response{
							Status:     http.StatusText(http.StatusOK),
							StatusCode: http.StatusOK,
						},
					},
				},
				importZonefile: actionResponse{
					err: errors.New("internal server error"),
				},
			},
			object: bulkChanges{
				zones: map[int64]*hcloud.Zone{
					1: {ID: 1, Name: "fastipletonis.eu"},
				},
				changes: map[int64]*zoneChanges{
					1: {
						creates: []*hetznerChangeCreate{
							{
								zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
								opts: hcloud.ZoneRRSetCreateOpts{
									Name: "ftp",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.1",
										},
									},
								},
							},
						},
						updates: []*hetznerChangeUpdate{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "2",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "www",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
								recordsOpts: &hcloud.ZoneRRSetSetRecordsOpts{
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
										{
											Value: "116.202.181.3",
										},
									},
								},
							},
						},
						deletes: []*hetznerChangeDelete{
							{
								rrset: &hcloud.ZoneRRSet{
									ID:   "3",
									Zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
									Name: "@",
									Type: hcloud.ZoneRRSetTypeA,
									TTL:  &ttl3600,
									Records: []hcloud.ZoneRRSetRecord{
										{
											Value: "116.202.181.2",
										},
									},
								},
							},
						},
					},
				},
			},
			input: struct {
				ctx  context.Context
				zone *hcloud.Zone
			}{
				ctx:  context.Background(),
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
			},
			expState: mockClientState{
				ExportZonefileCalled: true,
				ImportZonefileCalled: true,
			},
			expArgs: struct {
				zone *hcloud.Zone
				opts hcloud.ZoneImportZonefileOpts
			}{
				zone: &hcloud.Zone{ID: 1, Name: "fastipletonis.eu"},
				opts: hcloud.ZoneImportZonefileOpts{
					Zonefile: strings.Replace(changedZoneFile, oldSOA, todayMinSerialNumber(), 1),
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
