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
var (
	splitter = regexp.MustCompile(`\s+`)
	txtSplit = regexp.MustCompile(`"([^"\\]|(\\.))*"`)
)

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
	var rs rrset

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
	fmt.Fprintf(&zoneBuilder, "$ORIGIN\t%s\n", origin)
	fmt.Fprintf(&zoneBuilder, "$TTL\t%d\n", ttl)
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

// expandName expands a name to its FQDN.
func (z Zonefile) expandName(name string) string {
	if name == "@" {
		name = z.origin
	} else if !strings.HasSuffix(name, ".") {
		name += "." + z.origin
	}
	return name
}

// expandTarget expands a target to its FQDN.
func (z Zonefile) expandTarget(name string) string {
	if !strings.HasSuffix(name, ".") {
		name += "." + z.origin
	}
	return name
}

// parseARecord parses an A record.
func (z Zonefile) parseARecord(name string, ttl int, arg string) (*dns.A, error) {
	ip, err := netip.ParseAddr(arg)
	if err != nil {
		return nil, fmt.Errorf("cannot parse address %s: %w", arg, err)
	}
	if !ip.Is4() {
		return nil, fmt.Errorf("Address %s is not IPv4, unsupported for record type A", arg)
	}
	return &dns.A{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		A: rdata.A{
			Addr: ip,
		},
	}, nil
}

// parseAAAARecord parses an AAAA record.
func (z Zonefile) parseAAAARecord(name string, ttl int, arg string) (*dns.AAAA, error) {
	ip, err := netip.ParseAddr(arg)
	if err != nil {
		return nil, fmt.Errorf("cannot parse address %s: %w", arg, err)
	}
	if !ip.Is6() {
		return nil, fmt.Errorf("Address %s is not IPv6, unsupported for record type AAAA", arg)
	}
	return &dns.AAAA{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		AAAA: rdata.AAAA{
			Addr: ip,
		},
	}, nil
}

// parseCNAMERecord parses a CNAME record.
func (z Zonefile) parseCNAMERecord(name string, ttl int, arg string) (*dns.CNAME, error) {
	target := z.expandTarget(arg)
	return &dns.CNAME{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		CNAME: rdata.CNAME{
			Target: target,
		},
	}, nil
}

// parseTXTRecord parses a TXT record.
func (z Zonefile) parseTXTRecord(name string, ttl int, arg string) (*dns.TXT, error) {
	rows := txtSplit.FindAllString(arg, -1)
	if rows == nil {
		return nil, fmt.Errorf("invalid TXT record: %s", arg)
	}
	return &dns.TXT{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		TXT: rdata.TXT{
			Txt: rows,
		},
	}, nil
}

// parseNSRecord parses a NS record.
func (z Zonefile) parseNSRecord(name string, ttl int, arg string) (*dns.NS, error) {
	ns := z.expandTarget(arg)
	return &dns.NS{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		NS: rdata.NS{
			Ns: ns,
		},
	}, nil
}

// parseSRVRecord parses an SRV record.
func (z Zonefile) parseSRVRecord(name string, ttl int, arg string) (*dns.SRV, error) {
	srv := splitter.Split(arg, 5)
	if len(srv) != 4 {
		return nil, fmt.Errorf("values for SRV record %s cannot be decoded from \"%s\"", name, arg)
	}
	p, err := strconv.Atoi(srv[0])
	if err != nil {
		return nil, fmt.Errorf("cannot decode priority for SRV record %s from \"%s\"", name, srv[0])
	}
	w, err := strconv.Atoi(srv[1])
	if err != nil {
		return nil, fmt.Errorf("cannot decode weight for SRV record %s from \"%s\"", name, srv[1])
	}
	pt, err := strconv.Atoi(srv[2])
	if err != nil {
		return nil, fmt.Errorf("cannot decode port for SRV record %s from \"%s\"", name, srv[2])
	}
	target := z.expandTarget(srv[3])
	return &dns.SRV{
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
	}, nil
}

// parseMXRecord parses an MX record.
func (z Zonefile) parseMXRecord(name string, ttl int, arg string) (*dns.MX, error) {
	mx := splitter.Split(arg, 3)
	if len(mx) != 2 {
		return nil, fmt.Errorf("Values for MX record %s cannot be decoded from \"%s\"", name, arg)
	}
	p, err := strconv.Atoi(mx[0])
	if err != nil {
		return nil, fmt.Errorf("Cannot read preference for MX record %s from \"%s\"", name, mx[0])
	}
	exchange := z.expandTarget(mx[1])
	return &dns.MX{
		Hdr: dns.Header{
			Name:  name,
			TTL:   uint32(ttl),
			Class: dns.ClassINET,
		},
		MX: rdata.MX{
			Preference: uint16(p),
			Mx:         exchange,
		},
	}, nil
}

// parseRecord invokes the correct handler depending on the dnsType.
func (z Zonefile) parseRecord(dnsType uint16, name string, ttl int, arg string) (dns.RR, error) {
	switch dnsType {
	case dns.TypeA:
		return z.parseARecord(name, ttl, arg)
	case dns.TypeAAAA:
		return z.parseAAAARecord(name, ttl, arg)
	case dns.TypeCNAME:
		return z.parseCNAMERecord(name, ttl, arg)
	case dns.TypeTXT:
		return z.parseTXTRecord(name, ttl, arg)
	case dns.TypeNS:
		return z.parseNSRecord(name, ttl, arg)
	case dns.TypeSRV:
		return z.parseSRVRecord(name, ttl, arg)
	case dns.TypeMX:
		return z.parseMXRecord(name, ttl, arg)
	}
	return nil, errors.New("type not supported")
}

// AddRecord adds a new recordset.
func (z *Zonefile) AddRecord(recordType string, name string, ttl int, records []string) error {
	name = z.expandName(name)
	dnsType, ok := dns.StringToType[recordType]
	if !ok {
		return fmt.Errorf("record type %s is not recognized", recordType)
	}
	key := fmt.Sprintf(fmtKey, name, dnsType)
	if _, ok := z.records[key]; ok {
		return fmt.Errorf("cannot add a recordset for %s because it already exists", name)
	}
	rr := make(rrset, len(records))
	for i, rec := range records {
		a, err := z.parseRecord(dnsType, name, ttl, rec)
		if err != nil {
			return err
		}
		rr[i] = a
	}
	z.records[key] = rr
	return nil
}

// UpdateRecord updates an existing recordset
func (z *Zonefile) UpdateRecord(recordType string, name string, ttl int, records []string) error {
	name = z.expandName(name)
	dnsType, ok := dns.StringToType[recordType]
	if !ok {
		return fmt.Errorf("record type %s is not recognized", recordType)
	}
	key := fmt.Sprintf(fmtKey, name, dnsType)
	if _, ok := z.records[key]; !ok {
		return fmt.Errorf("cannot update recordset for %s because it does not exist", name)
	}
	rr := make(rrset, len(records))
	for i, rec := range records {
		a, err := z.parseRecord(dnsType, name, ttl, rec)
		if err != nil {
			return err
		}
		rr[i] = a
	}
	z.records[key] = rr
	return nil
}

// DeleteRecord deletes an existing recordset
func (z *Zonefile) DeleteRecord(recordType string, name string) error {
	name = z.expandName(name)
	dnsType, ok := dns.StringToType[recordType]
	if !ok {
		return fmt.Errorf("record type %s is not recognized", recordType)
	}
	key := fmt.Sprintf(fmtKey, name, dnsType)
	if _, ok := z.records[key]; !ok {
		return fmt.Errorf("cannot delete recordset %s of type %s because it does not exist", name, recordType)
	}
	delete(z.records, key)
	return nil
}
