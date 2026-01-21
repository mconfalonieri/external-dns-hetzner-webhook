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
$ORIGIN fastipletonis.eu.
$TTL 86400
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

// assertError checks if an error is thrown when expected.
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

func Test_updateSOA(t *testing.T) {
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

func Test_Export(t *testing.T) {
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
		expSN := strconv.Itoa(int(todayMaxSerialNumber() - 99))
		expFile := strings.Replace(exp.file, "2025112009", expSN, 1)
		expArray := strings.Split(expFile, "\n")
		array := strings.Split(file, "\n")
		assertError(t, exp.err, err)
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
