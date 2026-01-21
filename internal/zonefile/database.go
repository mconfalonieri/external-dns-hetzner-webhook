/*
 * Database - zonefile database.
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
	"fmt"
	"io"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
)

const (
	fmtKey = "%s|%d"
)

// Splitter for string components.
var splitter = regexp.MustCompile(`\s+`)

// rrset is an array of RRs
type rrset []dns.RR

// Zonefile stores the logical information from a zonefile for further
// manipulation. The following record types can be manipulated: A, AAAA, CNAME,
// NS, SRV, TXT. The other record types will be preserved.
type Zonefile struct {
	zoneName string
	records  map[string]rrset
	soaKey   string
	origin   string
	ttl      int
}

// GetOrigin returns the zonefile origin.
func (z Zonefile) GetOrigin() string {
	return z.origin
}

// GetTTL returns the zonefile TTL.
func (z Zonefile) GetTTL() int {
	return z.ttl
}

// getrrset gets or creates an rrset for the given key.
func getrrset(k string, m map[string]rrset) rrset {
	if r, ok := m[k]; ok {
		return r
	}
	m[k] = make(rrset, 0)
	return m[k]
}

// readRecords reads all the RR records from the file.
func readRecords(zp *dns.ZoneParser) (map[string]rrset, error) {
	records := make(map[string]rrset, 0)
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		k := fmt.Sprintf(fmtKey, rr.Header().Name, dns.RRToType(rr))
		set := getrrset(k, records)
		set = append(set, rr)
		records[k] = set
	}
	if len(records) == 0 {
		return nil, errors.New("cannot read records")
	}
	return records, nil
}

// NewZonefile creates a new logical zonefile. The parameters are a reader
// that will be used as a source and the zone name.
func NewZonefile(r io.Reader, zn string, ttl int) (*Zonefile, error) {
	origin := zn + "."
	file := zn + ".zone"
	zp := dns.NewZoneParser(r, origin, file)
	records, err := readRecords(zp)
	if err != nil {
		return nil, fmt.Errorf("cannot import zone %s: %w", zn, err)
	}

	return &Zonefile{
		zoneName: zn,
		records:  records,
		soaKey:   fmt.Sprintf(fmtKey, origin, dns.TypeSOA),
		origin:   origin,
		ttl:      ttl,
	}, nil
}

// updateSOASerialNumber increments the SOA serial number for export
func updateSOASerialNumber(sn uint32) (uint32, error) {
	soaSerialNumber, err := NewSOASerialNumber(fmt.Sprintf("%d", sn))
	if err != nil {
		return 0, err
	}
	err = soaSerialNumber.Inc()
	if err != nil {
		return 0, err
	}
	return soaSerialNumber.Uint32(), nil
}

// updateSOA finds the SOA record, updates its serial number and returns it.
func (z Zonefile) updateSOA() (*dns.SOA, error) {
	rs := make([]dns.RR, 1)

	if soas, ok := z.records[z.soaKey]; ok && len(soas) == 1 {
		rs = soas
	} else {
		return nil, fmt.Errorf("found %d SOA records instead of 1", len(soas))
	}

	rSOA := rs[0]
	soa, ok := rSOA.(*dns.SOA)
	if !ok {
		n := rSOA.Header().Name
		t := dns.RRToType(rSOA)
		return nil, fmt.Errorf("conversion error for SOA record (%s|%d)", n, t)
	}

	sn, err := updateSOASerialNumber(soa.Serial)
	if err != nil {
		return nil, err
	}
	soa.Serial = sn
	return soa, nil
}

// buildFile buils a zonefile from a set of records.
func buildFile(recs rrset, origin string, ttl int) string {
	var zoneBuilder strings.Builder
	fmt.Fprint(&zoneBuilder, ";; Created by external-dns-hetzner-webhook\n")
	fmt.Fprintf(&zoneBuilder, "$ORIGIN %s\n", origin)
	fmt.Fprintf(&zoneBuilder, "$TTL %d\n", ttl)
	for _, rr := range recs {
		s := rr.String()
		fmt.Fprintf(&zoneBuilder, "%s\n", s)
	}
	return zoneBuilder.String()
}

// Export exports the updated zonefile.
func (z Zonefile) Export() (string, error) {
	recs := make(rrset, 1)
	ttl := z.ttl
	soa, err := z.updateSOA()
	if err != nil {
		return "", fmt.Errorf("cannot export zonefile: %w", err)
	}
	if ttl <= 0 {
		ttl = int(soa.SOA.Minttl)
	}
	recs[0] = soa
	for k, slice := range z.records {
		if k == z.soaKey {
			continue
		}
		recs = append(recs, slice...)
	}
	file := buildFile(recs, z.origin, ttl)
	return file, nil
}

// AddARecord adds a new A recordset.
func (z *Zonefile) AddARecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeA)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for A record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, addr := range records {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return fmt.Errorf("cannot parse address %s, %w", addr, err)
		}
		if !ip.Is4() {
			return fmt.Errorf("Address %s is not IPv4, unsupported for record type A", addr)
		}
		r := &dns.A{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			A: rdata.A{
				Addr: ip,
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeARecord changes an existing A recordset
func (z *Zonefile) ChangeARecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeA)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for A record %s because it does not exist", name)
	}
	for i, addr := range records {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return fmt.Errorf("cannot parse address %s, %w", addr, err)
		}
		if !ip.Is4() {
			return fmt.Errorf("Address %s is not IPv4, unsupported for record type A", addr)
		}
		r := &dns.A{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			A: rdata.A{
				Addr: ip,
			},
		}
		rr[i] = r
	}
	return nil
}

// AddAAAARecord adds a new AAAA recordset.
func (z *Zonefile) AddAAAARecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeAAAA)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for AAAA record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, addr := range records {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return fmt.Errorf("cannot parse address %s, %w", addr, err)
		}
		if !ip.Is6() {
			return fmt.Errorf("Address %s is not IPv6, unsupported for record type AAAA", addr)
		}
		r := &dns.AAAA{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			AAAA: rdata.AAAA{
				Addr: ip,
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeAAAARecord changes an existing recordset.
func (z *Zonefile) ChangeAAAARecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeAAAA)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for AAAA record %s because it does not exist", name)
	}
	for i, addr := range records {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return fmt.Errorf("cannot parse address %s, %w", addr, err)
		}
		if !ip.Is6() {
			return fmt.Errorf("Address %s is not IPv6, unsupported for record type AAAA", addr)
		}
		r := &dns.AAAA{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			AAAA: rdata.AAAA{
				Addr: ip,
			},
		}
		rr[i] = r
	}
	return nil
}

// AddCNAMERecord adds a new CNAME recordset.
func (z *Zonefile) AddCNAMERecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeCNAME)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for CNAME record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, cname := range records {
		r := &dns.CNAME{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			CNAME: rdata.CNAME{
				Target: cname,
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeCNAMERecord changes an existing CNAME recordset
func (z *Zonefile) ChangeCNAMERecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeCNAME)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for CNAME record %s because it does not exist", name)
	}
	for i, cname := range records {
		r := &dns.CNAME{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			CNAME: rdata.CNAME{
				Target: cname,
			},
		}
		rr[i] = r
	}
	return nil
}

// AddTXTRecord adds a new TXT recordset.
func (z *Zonefile) AddTXTRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeTXT)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for TXT record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, txt := range records {
		r := &dns.TXT{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			TXT: rdata.TXT{
				Txt: []string{txt},
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeTXTRecord changes an existing TXT recordset
func (z *Zonefile) ChangeTXTRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeTXT)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for TXT record %s because it does not exist", name)
	}
	for i, txt := range records {
		r := &dns.TXT{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			TXT: rdata.TXT{
				Txt: []string{txt},
			},
		}
		rr[i] = r
	}
	return nil
}

// AddNSRecord adds a new NS recordset.
func (z *Zonefile) AddNSRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeNS)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for NS record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, ns := range records {
		r := &dns.NS{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			NS: rdata.NS{
				Ns: ns,
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeNSRecord changes an existing NS recordset
func (z *Zonefile) ChangeNSRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeNS)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for NS record %s because it does not exist", name)
	}
	for i, ns := range records {
		r := &dns.NS{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			NS: rdata.NS{
				Ns: ns,
			},
		}
		rr[i] = r
	}
	return nil
}

// AddSRVRecord adds a new SRV recordset.
func (z *Zonefile) AddSRVRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeSRV)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for SRV record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, v := range records {
		srv := splitter.Split(v, -1)
		if len(srv) != 4 {
			return fmt.Errorf("Values for SRV record %s cannot be decoded from \"%s\"", name, v)
		}
		p, err := strconv.Atoi(srv[0])
		if err != nil {
			return fmt.Errorf("Cannot decode priority for SRV record %s from \"%s\"", name, srv[0])
		}
		w, err := strconv.Atoi(srv[1])
		if err != nil {
			return fmt.Errorf("Cannot decode weight for SRV record %s from \"%s\"", name, srv[1])
		}
		pt, err := strconv.Atoi(srv[2])
		if err != nil {
			return fmt.Errorf("Cannot decode port for SRV record %s from \"%s\"", name, srv[2])
		}
		target := srv[3]
		r := &dns.SRV{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			SRV: rdata.SRV{
				Priority: uint16(p),
				Weight:   uint16(w),
				Port:     uint16(pt),
				Target:   target,
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeSRVRecord changes an existing SRV recordset
func (z *Zonefile) ChangeSRVRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeSRV)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for SRV record %s because it does not exist", name)
	}
	for i, v := range records {
		srv := splitter.Split(v, -1)
		if len(srv) != 4 {
			return fmt.Errorf("Values for SRV record %s cannot be decoded from \"%s\"", name, v)
		}
		p, err := strconv.Atoi(srv[0])
		if err != nil {
			return fmt.Errorf("Cannot decode priority for SRV record %s from \"%s\"", name, srv[0])
		}
		w, err := strconv.Atoi(srv[1])
		if err != nil {
			return fmt.Errorf("Cannot decode weight for SRV record %s from \"%s\"", name, srv[1])
		}
		pt, err := strconv.Atoi(srv[2])
		if err != nil {
			return fmt.Errorf("Cannot decode port for SRV record %s from \"%s\"", name, srv[2])
		}
		target := srv[3]
		r := &dns.SRV{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			SRV: rdata.SRV{
				Priority: uint16(p),
				Weight:   uint16(w),
				Port:     uint16(pt),
				Target:   target,
			},
		}
		rr[i] = r
	}
	return nil
}

// AddMXRecord adds a new MX recordset.
func (z *Zonefile) AddMXRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeMX)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for CNAME record %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, v := range records {
		mx := splitter.Split(v, -1)
		if len(mx) != 2 {
			return fmt.Errorf("Values for MX record %s cannot be decoded from \"%s\"", name, v)
		}
		p, err := strconv.Atoi(mx[0])
		if err != nil {
			return fmt.Errorf("Cannot read preference for MX record %s from \"%s\"", name, v)
		}
		r := &dns.MX{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			MX: rdata.MX{
				Preference: uint16(p),
				Mx:         mx[1],
			},
		}
		rr[i] = r
	}
	z.records[key] = rr
	return nil
}

// ChangeMXRecord changes an existing MX recordset
func (z *Zonefile) ChangeMXRecord(name string, ttl int, records []string) error {
	key := fmt.Sprintf(fmtKey, name, dns.TypeCNAME)
	rr, ok := z.records[key]
	if !ok {
		return fmt.Errorf("cannot change recordset for MX record %s because it does not exist", name)
	}
	for i, v := range records {
		mx := splitter.Split(v, -1)
		if len(mx) != 2 {
			return fmt.Errorf("Values for MX record %s cannot be decoded from \"%s\"", name, v)
		}
		p, err := strconv.Atoi(mx[0])
		if err != nil {
			return fmt.Errorf("Cannot read preference for MX record %s from \"%s\"", name, v)
		}
		r := &dns.MX{
			Hdr: dns.Header{
				Name:  name,
				TTL:   uint32(ttl),
				Class: dns.ClassINET,
			},
			MX: rdata.MX{
				Preference: uint16(p),
				Mx:         mx[1],
			},
		}
		rr[i] = r
	}
	return nil
}
