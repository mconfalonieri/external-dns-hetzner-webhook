/*
 * Database - Test suite.
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
	"io"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
	"github.com/stretchr/testify/assert"
)

const (
	testMiniZonefile = `;; Exported on 2026-01-19T21:39:41Z
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
	testExportedZonefile = `;; Created by external-dns-hetzner-webhook
$ORIGIN	fastipletonis.eu.
$TTL	86400
fastipletonis.eu.	3600	IN	SOA	hydrogen.ns.hetzner.com. dns.hetzner.com. 2025112009 86400 10800 3600000 3600
fastipletonis.eu.	3600	IN	NS	helium.ns.hetzner.de.
fastipletonis.eu.	3600	IN	NS	hydrogen.ns.hetzner.com.
fastipletonis.eu.	3600	IN	NS	oxygen.ns.hetzner.com.
fastipletonis.eu.	3600	IN	CAA	128 issue "letsencrypt.org"
fastipletonis.eu.	3600	IN	A	116.202.181.2
www.fastipletonis.eu.	3600	IN	A	116.202.181.2
`
	testZone         = "fastipletonis.eu"
	testOrigin       = "fastipletonis.eu."
	testZonefileName = "fastipletonis.eu.zone"
	testSoaKey       = "fastipletonis.eu.|6"
	testTTL          = 86400
)

// sortRows sorts the file rows for comparison.
func sortRows(file string) []string {
	array := strings.Split(file, "\n")
	slices.Sort(array)
	return array
}

// assertError checks if an error is thrown when expected. Returns true if an
// error is expected.
func assertError(t *testing.T, expected, actual error) bool {
	var expError bool
	if expected == nil {
		assert.Nil(t, actual)
		expError = false
	} else {
		assert.EqualError(t, actual, expected.Error())
		expError = true
	}
	return expError
}

// todayMaxSerialNumber returns today's maximum serial number.
func todayMaxSerialNumber() uint32 {
	strSerialNumber := time.Now().Format(fmtSOADate) + "99"
	serialNumber, _ := strconv.Atoi(strSerialNumber)
	return uint32(serialNumber)
}

func Test_getrrset(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			k string
			m map[string]rrset
		}
		expected struct {
			mk   rrset
			mlen int
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		actual := getrrset(inp.k, inp.m)
		actualLen := len(inp.m)
		assert.EqualValues(t, exp.mk, actual)
		assert.Equal(t, exp.mlen, actualLen)
	}

	testCases := []testCase{
		{
			name: "rrset exists",
			input: struct {
				k string
				m map[string]rrset
			}{
				k: testSoaKey,
				m: map[string]rrset{testSoaKey: {&dns.SOA{}}},
			},
			expected: struct {
				mk   rrset
				mlen int
			}{
				mk:   rrset{&dns.SOA{}},
				mlen: 1,
			},
		},
		{
			name: "rrset does not exist",
			input: struct {
				k string
				m map[string]rrset
			}{
				k: testSoaKey,
				m: map[string]rrset{},
			},
			expected: struct {
				mk   rrset
				mlen int
			}{
				mk:   rrset{},
				mlen: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_readRecords(t *testing.T) {
	type testCase struct {
		name     string
		input    *dns.ZoneParser
		expected struct {
			records map[string]rrset
			err     error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		records, err := readRecords(tc.input)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.records, records)
	}

	testCases := []testCase{
		{
			name:  "no records",
			input: dns.NewZoneParser(strings.NewReader(""), testOrigin, testZonefileName),
			expected: struct {
				records map[string]rrset
				err     error
			}{
				records: nil,
				err:     errors.New("cannot read records"),
			},
		},
		{
			name:  "valid records",
			input: dns.NewZoneParser(strings.NewReader(testMiniZonefile), testOrigin, testZonefileName),
			expected: struct {
				records map[string]rrset
				err     error
			}{
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
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

func Test_NewZonefile(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			r   io.Reader
			zn  string
			ttl int
		}
		expected struct {
			z   *Zonefile
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		exp := tc.expected
		z, err := NewZonefile(inp.r, inp.zn, inp.ttl)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.z, z)
	}

	testCases := []testCase{
		{
			name: "error",
			input: struct {
				r   io.Reader
				zn  string
				ttl int
			}{
				r:   strings.NewReader(""),
				zn:  testZone,
				ttl: 3600,
			},
			expected: struct {
				z   *Zonefile
				err error
			}{
				z:   nil,
				err: errors.New("cannot import zone fastipletonis.eu: cannot read records"),
			},
		},
		{
			name: "valid records",
			input: struct {
				r   io.Reader
				zn  string
				ttl int
			}{
				r:   strings.NewReader(testMiniZonefile),
				zn:  testZone,
				ttl: 3600,
			},
			expected: struct {
				z   *Zonefile
				err error
			}{
				z: &Zonefile{
					zoneName: testZone,
					records: map[string]rrset{
						"fastipletonis.eu.|6": {
							&dns.SOA{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								SOA: rdata.SOA{
									Ns:      "hydrogen.ns.hetzner.com.",
									Mbox:    "dns.hetzner.com.",
									Serial:  2025112009,
									Refresh: 86400,
									Retry:   10800,
									Expire:  3600000,
									Minttl:  3600,
								},
							},
						},
						"fastipletonis.eu.|2": {
							&dns.NS{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								NS: rdata.NS{
									Ns: "helium.ns.hetzner.de.",
								},
							},
							&dns.NS{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								NS: rdata.NS{
									Ns: "hydrogen.ns.hetzner.com.",
								},
							},
							&dns.NS{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								NS: rdata.NS{
									Ns: "oxygen.ns.hetzner.com.",
								},
							},
						},
						"fastipletonis.eu.|257": {
							&dns.CAA{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								CAA: rdata.CAA{
									Flag:  128,
									Tag:   "issue",
									Value: "letsencrypt.org",
								},
							},
						},
						"fastipletonis.eu.|1": {
							&dns.A{
								Hdr: dns.Header{
									Name:  "fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								A: rdata.A{
									Addr: netip.MustParseAddr("116.202.181.2"),
								},
							},
						},
						"www.fastipletonis.eu.|1": {
							&dns.A{
								Hdr: dns.Header{
									Name:  "www.fastipletonis.eu.",
									TTL:   3600,
									Class: dns.ClassINET,
								},
								A: rdata.A{
									Addr: netip.MustParseAddr("116.202.181.2"),
								},
							},
						},
					},
					soaKey: testSoaKey,
					ttl:    3600,
					origin: testOrigin,
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

func Test_updateSOASerialNumber(t *testing.T) {
	type testCase struct {
		name     string
		input    uint32
		expected struct {
			sn  uint32
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		sn, err := updateSOASerialNumber(tc.input)
		assertError(t, exp.err, err)
		assert.Equal(t, exp.sn, sn)
	}

	testCases := []testCase{
		{
			name:  "unreadable serial number",
			input: 0,
			expected: struct {
				sn  uint32
				err error
			}{
				sn:  0,
				err: errors.New("serial number \"0\" is unsupported"),
			},
		},
		{
			name:  "increment error",
			input: todayMaxSerialNumber(),
			expected: struct {
				sn  uint32
				err error
			}{
				sn:  0,
				err: errors.New("cannot increment version as it is 99"),
			},
		},
		{
			name:  "valid serial number",
			input: todayMaxSerialNumber() - 1,
			expected: struct {
				sn  uint32
				err error
			}{
				sn:  todayMaxSerialNumber(),
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

func Test_Zonefile_updateSOA(t *testing.T) {
	type testCase struct {
		name     string
		object   *Zonefile
		expected struct {
			soa *dns.SOA
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		exp := tc.expected
		obj := tc.object
		soa, err := obj.updateSOA()
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.soa, soa)
	}

	testCases := []testCase{
		{
			name: "no soa record",
			object: &Zonefile{
				zoneName: testZone,
				records:  map[string]rrset{},
				soaKey:   testSoaKey,
				ttl:      3600,
			},
			expected: struct {
				soa *dns.SOA
				err error
			}{
				soa: nil,
				err: errors.New("found 0 SOA records instead of 1"),
			},
		},
		{
			name: "valid update",
			object: &Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    3600,
			},
			expected: struct {
				soa *dns.SOA
				err error
			}{
				soa: &dns.SOA{
					Hdr: dns.Header{
						Name:  "fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					SOA: rdata.SOA{
						Ns:      "hydrogen.ns.hetzner.com.",
						Mbox:    "dns.hetzner.com.",
						Serial:  todayMaxSerialNumber() - 99,
						Refresh: 86400,
						Retry:   10800,
						Expire:  3600000,
						Minttl:  3600,
					},
				},
			},
		},
		{
			name: "invalid version",
			object: &Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  todayMaxSerialNumber(),
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    3600,
			},
			expected: struct {
				soa *dns.SOA
				err error
			}{
				err: errors.New("cannot increment version as it is 99"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_buildFile(t *testing.T) {
	type testCase struct {
		name  string
		input struct {
			recs   rrset
			origin string
			ttl    int
		}
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		inp := tc.input
		actual := buildFile(inp.recs, inp.origin, inp.ttl)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "create zonefile",
			input: struct {
				recs   rrset
				origin string
				ttl    int
			}{
				recs: rrset{
					&dns.SOA{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						SOA: rdata.SOA{
							Ns:      "hydrogen.ns.hetzner.com.",
							Mbox:    "dns.hetzner.com.",
							Serial:  2025112009,
							Refresh: 86400,
							Retry:   10800,
							Expire:  3600000,
							Minttl:  3600,
						},
					},
					&dns.NS{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						NS: rdata.NS{
							Ns: "helium.ns.hetzner.de.",
						},
					},
					&dns.NS{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						NS: rdata.NS{
							Ns: "hydrogen.ns.hetzner.com.",
						},
					},
					&dns.NS{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						NS: rdata.NS{
							Ns: "oxygen.ns.hetzner.com.",
						},
					},
					&dns.CAA{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						CAA: rdata.CAA{
							Flag:  128,
							Tag:   "issue",
							Value: "letsencrypt.org",
						},
					},
					&dns.A{
						Hdr: dns.Header{
							Name:  "fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						A: rdata.A{
							Addr: netip.MustParseAddr("116.202.181.2"),
						},
					},
					&dns.A{
						Hdr: dns.Header{
							Name:  "www.fastipletonis.eu.",
							TTL:   3600,
							Class: dns.ClassINET,
						},
						A: rdata.A{
							Addr: netip.MustParseAddr("116.202.181.2"),
						},
					},
				},
				origin: testOrigin,
				ttl:    testTTL,
			},
			expected: testExportedZonefile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_Export(t *testing.T) {
	type testCase struct {
		name     string
		object   Zonefile
		expected struct {
			file string
			err  error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		exp := tc.expected
		file, err := obj.Export()
		if assertError(t, exp.err, err) {
			assert.Equal(t, "", file)
			return
		}
		expSN := strconv.Itoa(int(todayMaxSerialNumber() - 99))
		expFile := strings.Replace(exp.file, "2025112009", expSN, 1)
		expArray := sortRows(expFile)
		array := sortRows(file)
		assert.EqualValues(t, expArray, array)
	}

	testCases := []testCase{
		{
			name: "create zonefile",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			expected: struct {
				file string
				err  error
			}{
				file: testExportedZonefile,
				err:  nil,
			},
		},
		{
			name: "soa update error",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  todayMaxSerialNumber(),
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			expected: struct {
				file string
				err  error
			}{
				file: "",
				err:  errors.New("cannot export zonefile: cannot increment version as it is 99"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_expandName(t *testing.T) {
	type testCase struct {
		name     string
		object   Zonefile
		input    string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.expandName(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "expand root",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input:    "@",
			expected: "fastipletonis.eu.",
		},
		{
			name: "expand relative hostname",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input:    "www",
			expected: "www.fastipletonis.eu.",
		},
		{
			name: "no expansion",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input:    "www.example.org.",
			expected: "www.example.org.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_expandTarget(t *testing.T) {
	type testCase struct {
		name     string
		object   Zonefile
		input    string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		actual := obj.expandName(tc.input)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "expand relative hostname",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input:    "www",
			expected: "www.fastipletonis.eu.",
		},
		{
			name: "no expansion",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input:    "www.example.org.",
			expected: "www.example.org.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_parseARecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			a   *dns.A
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		a, err := obj.parseARecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.a, a)
	}

	testCases := []testCase{
		{
			name: "parsed address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www.fastipletonis.eu.",
				ttl:    3600,
				record: "10.0.0.1",
			},
			expected: struct {
				a   *dns.A
				err error
			}{
				a: &dns.A{
					Hdr: dns.Header{
						Name:  "www.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					A: rdata.A{
						Addr: netip.MustParseAddr("10.0.0.1"),
					},
				},
				err: nil,
			},
		},
		{
			name: "error unparseable address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www",
				ttl:    3600,
				record: "localhost",
			},
			expected: struct {
				a   *dns.A
				err error
			}{
				a:   nil,
				err: errors.New("cannot parse address localhost: ParseAddr(\"localhost\"): unable to parse IP"),
			},
		},
		{
			name: "error no ipv4 address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www",
				ttl:    3600,
				record: "2001:db8:85a3:0:0:8a2e:370:7334",
			},
			expected: struct {
				a   *dns.A
				err error
			}{
				a:   nil,
				err: errors.New("Address 2001:db8:85a3:0:0:8a2e:370:7334 is not IPv4, unsupported for record type A"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_parseAAAARecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			aaaa *dns.AAAA
			err  error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		aaaa, err := obj.parseAAAARecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.aaaa, aaaa)
	}

	testCases := []testCase{
		{
			name: "parsed address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www.fastipletonis.eu.",
				ttl:    3600,
				record: "2001:db8:85a3:0:0:8a2e:370:7334",
			},
			expected: struct {
				aaaa *dns.AAAA
				err  error
			}{
				aaaa: &dns.AAAA{
					Hdr: dns.Header{
						Name:  "www.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					AAAA: rdata.AAAA{
						Addr: netip.MustParseAddr("2001:db8:85a3:0:0:8a2e:370:7334"),
					},
				},
				err: nil,
			},
		},
		{
			name: "error unparseable address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www",
				ttl:    3600,
				record: "localhost",
			},
			expected: struct {
				aaaa *dns.AAAA
				err  error
			}{
				aaaa: nil,
				err:  errors.New("cannot parse address localhost: ParseAddr(\"localhost\"): unable to parse IP"),
			},
		},
		{
			name: "error no ipv6 address",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www",
				ttl:    3600,
				record: "10.0.0.1",
			},
			expected: struct {
				aaaa *dns.AAAA
				err  error
			}{
				aaaa: nil,
				err:  errors.New("Address 10.0.0.1 is not IPv6, unsupported for record type AAAA"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_parseCNAMERecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			cname *dns.CNAME
			err   error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		cname, err := obj.parseCNAMERecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.cname, cname)
	}

	testCases := []testCase{
		{
			name: "relative target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www.fastipletonis.eu.",
				ttl:    3600,
				record: "ftp",
			},
			expected: struct {
				cname *dns.CNAME
				err   error
			}{
				cname: &dns.CNAME{
					Hdr: dns.Header{
						Name:  "www.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					CNAME: rdata.CNAME{
						Target: "ftp.fastipletonis.eu.",
					},
				},
				err: nil,
			},
		},
		{
			name: "absolute target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "www.fastipletonis.eu.",
				ttl:    3600,
				record: "ftp.example.org.",
			},
			expected: struct {
				cname *dns.CNAME
				err   error
			}{
				cname: &dns.CNAME{
					Hdr: dns.Header{
						Name:  "www.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					CNAME: rdata.CNAME{
						Target: "ftp.example.org.",
					},
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

func Test_Zonefile_parseTXTRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			txt *dns.TXT
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		txt, err := obj.parseTXTRecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.txt, txt)
	}

	testCases := []testCase{
		{
			name: "single line",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "reg.fastipletonis.eu.",
				ttl:    3600,
				record: `"test=value"`,
			},
			expected: struct {
				txt *dns.TXT
				err error
			}{
				txt: &dns.TXT{
					Hdr: dns.Header{
						Name:  "reg.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					TXT: rdata.TXT{
						Txt: []string{"test=value"},
					},
				},
				err: nil,
			},
		},
		{
			name: "single line with quotes",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "reg.fastipletonis.eu.",
				ttl:    3600,
				record: `"test=\"value\""`,
			},
			expected: struct {
				txt *dns.TXT
				err error
			}{
				txt: &dns.TXT{
					Hdr: dns.Header{
						Name:  "reg.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					TXT: rdata.TXT{
						Txt: []string{`test="value"`},
					},
				},
				err: nil,
			},
		},
		{
			name: "multiple lines",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "reg.fastipletonis.eu.",
				ttl:    3600,
				record: `"test=value" "prod=value"`,
			},
			expected: struct {
				txt *dns.TXT
				err error
			}{
				txt: &dns.TXT{
					Hdr: dns.Header{
						Name:  "reg.fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					TXT: rdata.TXT{
						Txt: []string{"test=value", "prod=value"},
					},
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

func Test_Zonefile_parseNSRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			ns  *dns.NS
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		ns, err := obj.parseNSRecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.ns, ns)
	}

	testCases := []testCase{
		{
			name: "relative target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "fastipletonis.eu.",
				ttl:    3600,
				record: "ns1",
			},
			expected: struct {
				ns  *dns.NS
				err error
			}{
				ns: &dns.NS{
					Hdr: dns.Header{
						Name:  "fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					NS: rdata.NS{
						Ns: "ns1.fastipletonis.eu.",
					},
				},
				err: nil,
			},
		},
		{
			name: "absolute target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "fastipletonis.eu.",
				ttl:    3600,
				record: "ns.example.org.",
			},
			expected: struct {
				ns  *dns.NS
				err error
			}{
				ns: &dns.NS{
					Hdr: dns.Header{
						Name:  "fastipletonis.eu.",
						TTL:   3600,
						Class: dns.ClassINET,
					},
					NS: rdata.NS{
						Ns: "ns.example.org.",
					},
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

func Test_Zonefile_parseSRVRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			srv *dns.SRV
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		srv, err := obj.parseSRVRecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.srv, srv)
	}

	testCases := []testCase{
		{
			name: "parsed with relative target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "10 0 25565 minecraft",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				srv: &dns.SRV{
					Hdr: dns.Header{
						Name:  "_minecraft._tcp.fastipletonis.eu.",
						Class: dns.ClassINET,
						TTL:   3600,
					},
					SRV: rdata.SRV{
						Priority: 10,
						Weight:   0,
						Port:     25565,
						Target:   "minecraft.fastipletonis.eu.",
					},
				},
			},
		},
		{
			name: "parsed with absolute target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "10 0 25565 minecraft.example.org.",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				srv: &dns.SRV{
					Hdr: dns.Header{
						Name:  "_minecraft._tcp.fastipletonis.eu.",
						Class: dns.ClassINET,
						TTL:   3600,
					},
					SRV: rdata.SRV{
						Priority: 10,
						Weight:   0,
						Port:     25565,
						Target:   "minecraft.example.org.",
					},
				},
			},
		},
		{
			name: "unparseable record",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "IN 10 0 25565 minecraft",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				err: errors.New("values for SRV record _minecraft._tcp.fastipletonis.eu. cannot be decoded from \"IN 10 0 25565 minecraft\""),
			},
		},
		{
			name: "unparseable priority",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "IN 10 25565 minecraft",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				err: errors.New("cannot decode priority for SRV record _minecraft._tcp.fastipletonis.eu. from \"IN\""),
			},
		},
		{
			name: "unparseable weight",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "10 W 25565 minecraft",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				err: errors.New("cannot decode weight for SRV record _minecraft._tcp.fastipletonis.eu. from \"W\""),
			},
		},
		{
			name: "unparseable port",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "10 0 PORT minecraft",
			},
			expected: struct {
				srv *dns.SRV
				err error
			}{
				err: errors.New("cannot decode port for SRV record _minecraft._tcp.fastipletonis.eu. from \"PORT\""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_parseMXRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			name   string
			ttl    int
			record string
		}
		expected struct {
			mx  *dns.MX
			err error
		}
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		exp := tc.expected
		mx, err := obj.parseMXRecord(inp.name, inp.ttl, inp.record)
		assertError(t, exp.err, err)
		assert.EqualValues(t, exp.mx, mx)
	}

	testCases := []testCase{
		{
			name: "parsed with relative target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "fastipletonis.eu.",
				ttl:    3600,
				record: "10 mbox",
			},
			expected: struct {
				mx  *dns.MX
				err error
			}{
				mx: &dns.MX{
					Hdr: dns.Header{
						Name:  "fastipletonis.eu.",
						Class: dns.ClassINET,
						TTL:   3600,
					},
					MX: rdata.MX{
						Preference: 10,
						Mx:         "mbox.fastipletonis.eu.",
					},
				},
			},
		},
		{
			name: "parsed with absolute target",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "fastipletonis.eu.",
				ttl:    3600,
				record: "10 mbox.example.org.",
			},
			expected: struct {
				mx  *dns.MX
				err error
			}{
				mx: &dns.MX{
					Hdr: dns.Header{
						Name:  "fastipletonis.eu.",
						Class: dns.ClassINET,
						TTL:   3600,
					},
					MX: rdata.MX{
						Preference: 10,
						Mx:         "mbox.example.org.",
					},
				},
			},
		},
		{
			name: "unparseable record",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "fastipletonis.eu.",
				ttl:    3600,
				record: "mbox",
			},
			expected: struct {
				mx  *dns.MX
				err error
			}{
				err: errors.New("Values for MX record fastipletonis.eu. cannot be decoded from \"mbox\""),
			},
		},
		{
			name: "unparseable preference",
			object: Zonefile{
				origin: "fastipletonis.eu.",
			},
			input: struct {
				name   string
				ttl    int
				record string
			}{
				name:   "_minecraft._tcp.fastipletonis.eu.",
				ttl:    3600,
				record: "PRI mbox",
			},
			expected: struct {
				mx  *dns.MX
				err error
			}{
				err: errors.New("Cannot read preference for MX record _minecraft._tcp.fastipletonis.eu. from \"PRI\""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_AddRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			recordType string
			name       string
			ttl        int
			records    []string
		}
		expected  error
		expObject Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		actual := obj.AddRecord(inp.recordType, inp.name, inp.ttl, inp.records)
		assertError(t, tc.expected, actual)
		assert.EqualValues(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "add single record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "ftp",
				ttl:        3600,
				records:    []string{"116.202.181.8"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"ftp.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "add multiple records",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "ftp",
				ttl:        3600,
				records:    []string{"116.202.181.8", "116.202.181.9"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"ftp.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.9"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "add single root record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "@",
				ttl:        3600,
				records:    []string{"116.202.181.2"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "add multiple root records",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "@",
				ttl:        3600,
				records:    []string{"116.202.181.2", "116.202.181.3"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.3"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error existing record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "www",
				ttl:        3600,
				records:    []string{"116.202.181.8"},
			},
			expected: errors.New("cannot add a recordset for www.fastipletonis.eu. because it already exists"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error unrecognized type",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "IPP",
				name:       "ftp",
				ttl:        3600,
				records:    []string{"127.0.0.1"},
			},
			expected: errors.New("record type IPP is not recognized"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_UpdateRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			recordType string
			name       string
			ttl        int
			records    []string
		}
		expected  error
		expObject Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		actual := obj.UpdateRecord(inp.recordType, inp.name, inp.ttl, inp.records)
		assertError(t, tc.expected, actual)
		assert.EqualValues(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "update single record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "www",
				ttl:        3600,
				records:    []string{"116.202.181.8"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "update with multiple records",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "www",
				ttl:        3600,
				records:    []string{"116.202.181.8", "116.202.181.9"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.9"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "update single root record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "@",
				ttl:        3600,
				records:    []string{"116.202.181.8"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "update multiple root addresses",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "@",
				ttl:        3600,
				records:    []string{"116.202.181.2", "116.202.181.3"},
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.3"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error missing record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "A",
				name:       "www",
				ttl:        3600,
				records:    []string{"116.202.181.8"},
			},
			expected: errors.New("cannot update recordset for www.fastipletonis.eu. because it does not exist"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error unrecognized type",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
				ttl        int
				records    []string
			}{
				recordType: "IPP",
				name:       "www",
				ttl:        3600,
				records:    []string{"localhost"},
			},
			expected: errors.New("record type IPP is not recognized"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Zonefile_DeleteRecord(t *testing.T) {
	type testCase struct {
		name   string
		object Zonefile
		input  struct {
			recordType string
			name       string
		}
		expected  error
		expObject Zonefile
	}

	run := func(t *testing.T, tc testCase) {
		obj := tc.object
		inp := tc.input
		actual := obj.DeleteRecord(inp.recordType, inp.name)
		assertError(t, tc.expected, actual)
		assert.EqualValues(t, tc.expObject, obj)
	}

	testCases := []testCase{
		{
			name: "delete single record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"ftp.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "A",
				name:       "ftp",
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "delete multiple records",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"ftp.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.8"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "ftp.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.9"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "A",
				name:       "ftp",
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "delete single root record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "A",
				name:       "@",
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "delete multiple root records",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.3"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "A",
				name:       "@",
			},
			expected: nil,
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error non existing record",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "A",
				name:       "ftp",
			},
			expected: errors.New("cannot delete recordset ftp.fastipletonis.eu. of type A because it does not exist"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
		{
			name: "error unrecognized type",
			object: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
			input: struct {
				recordType string
				name       string
			}{
				recordType: "IPP",
				name:       "www",
			},
			expected: errors.New("record type IPP is not recognized"),
			expObject: Zonefile{
				zoneName: testZone,
				records: map[string]rrset{
					"fastipletonis.eu.|6": {
						&dns.SOA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							SOA: rdata.SOA{
								Ns:      "hydrogen.ns.hetzner.com.",
								Mbox:    "dns.hetzner.com.",
								Serial:  2025112009,
								Refresh: 86400,
								Retry:   10800,
								Expire:  3600000,
								Minttl:  3600,
							},
						},
					},
					"fastipletonis.eu.|2": {
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "helium.ns.hetzner.de.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "hydrogen.ns.hetzner.com.",
							},
						},
						&dns.NS{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							NS: rdata.NS{
								Ns: "oxygen.ns.hetzner.com.",
							},
						},
					},
					"fastipletonis.eu.|257": {
						&dns.CAA{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							CAA: rdata.CAA{
								Flag:  128,
								Tag:   "issue",
								Value: "letsencrypt.org",
							},
						},
					},
					"fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
					"www.fastipletonis.eu.|1": {
						&dns.A{
							Hdr: dns.Header{
								Name:  "www.fastipletonis.eu.",
								TTL:   3600,
								Class: dns.ClassINET,
							},
							A: rdata.A{
								Addr: netip.MustParseAddr("116.202.181.2"),
							},
						},
					},
				},
				soaKey: testSoaKey,
				ttl:    86400,
				origin: testOrigin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
