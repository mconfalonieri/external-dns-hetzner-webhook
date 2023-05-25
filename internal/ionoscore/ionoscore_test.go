package ionoscore

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/plan"

	sdk "github.com/ionos-developer/dns-sdk-go"
	"github.com/stretchr/testify/require"
)

type mockDnsService struct {
	testErrorReturned bool
}

func TestNewProvider(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	t.Setenv("IONOS_API_KEY", "1")
	p, err := NewProvider(endpoint.NewDomainFilter([]string{"a.de."}), true)
	if err != nil {
		t.Errorf("should not fail, %s", err)
	}

	require.Equal(t, true, p.dryRun)
	require.Equal(t, true, p.domainFilter.IsConfigured())
	require.Equal(t, false, p.domainFilter.Match("b.de."))

	p, err = NewProvider(endpoint.DomainFilter{}, false)

	if err != nil {
		t.Errorf("should not fail, %s", err)
	}

	require.Equal(t, false, p.dryRun)
	require.Equal(t, false, p.domainFilter.IsConfigured())
	require.Equal(t, true, p.domainFilter.Match("a.de."))

	_ = os.Unsetenv("IONOS_API_KEY")
	_, err = NewProvider(endpoint.DomainFilter{}, true)

	if err == nil {
		t.Errorf("expected to fail")
	}
}

func TestRecords(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()

	provider := &Provider{client: mockDnsService{testErrorReturned: false}}
	endpoints, err := provider.Records(ctx)
	if err != nil {
		t.Errorf("should not fail, %s", err)
	}
	require.Equal(t, 5, len(endpoints))

	provider = &Provider{client: mockDnsService{testErrorReturned: true}}
	_, err = provider.Records(ctx)

	if err == nil {
		t.Errorf("expected to fail, %s", err)
	}
}

func TestApplyChanges(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()

	provider := &Provider{client: mockDnsService{testErrorReturned: false}}
	err := provider.ApplyChanges(ctx, changes())
	if err != nil {
		t.Errorf("should not fail, %s", err)
	}

	// 3 records must be deleted
	require.Equal(t, deletedRecords["b"], []string{"6"})
	sort.Strings(deletedRecords["a"])
	require.Equal(t, deletedRecords["a"], []string{"1", "2"})
	// 3 records must be created
	if !isRecordCreated("a", "a.de", sdk.A, "3.3.3.3", 2000) {
		t.Errorf("Record a.de A 3.3.3.3 not created")
	}
	if !isRecordCreated("a", "a.de", sdk.A, "4.4.4.4", 2000) {
		t.Errorf("Record a.de A 4.4.4.4 not created")
	}
	if !isRecordCreated("a", "new.a.de", sdk.CNAME, "a.de", 0) {
		t.Errorf("Record new.a.de CNAME a.de not created")
	}

	provider = &Provider{client: mockDnsService{testErrorReturned: true}}
	err = provider.ApplyChanges(ctx, nil)

	if err == nil {
		t.Errorf("expected to fail, %s", err)
	}
}

func (m mockDnsService) GetZones(ctx context.Context) ([]sdk.Zone, error) {
	if m.testErrorReturned {
		return nil, fmt.Errorf("GetZones failed")
	}

	a := sdk.NewZone()
	a.SetId("a")
	a.SetName("a.de")

	b := sdk.NewZone()
	b.SetId("b")
	b.SetName("b.de")

	return []sdk.Zone{*a, *b}, nil
}

func (m mockDnsService) GetZone(ctx context.Context, zoneId string) (*sdk.CustomerZone, error) {
	if m.testErrorReturned {
		return nil, fmt.Errorf("GetZone failed")
	}

	zoneName := zoneIdToZoneName[zoneId]
	zone := sdk.NewCustomerZone()
	zone.Id = &zoneId
	zone.Name = &zoneName
	if zoneName == "a.de" {
		zone.Records = []sdk.RecordResponse{
			record(1, "a.de", sdk.A, "1.1.1.1", 1000),
			record(2, "a.de", sdk.A, "2.2.2.2", 1000),
			record(3, "cname.a.de", sdk.CNAME, "cname.de", 1000),
			record(4, "aaaa.a.de", sdk.AAAA, "1::", 1000),
			record(5, "aaaa.a.de", sdk.AAAA, "2::", 2000),
		}
	} else {
		zone.Records = []sdk.RecordResponse{record(6, "b.de", sdk.A, "5.5.5.5", 1000)}
	}

	return zone, nil
}

func (m mockDnsService) CreateRecords(ctx context.Context, zoneId string, records []sdk.Record) error {
	createdRecords[zoneId] = append(createdRecords[zoneId], records...)
	return nil
}

func (m mockDnsService) DeleteRecord(ctx context.Context, zoneId string, recordId string) error {
	deletedRecords[zoneId] = append(deletedRecords[zoneId], recordId)
	return nil
}

func record(id int, name string, recordType sdk.RecordTypes, content string, ttl int32) sdk.RecordResponse {
	r := sdk.NewRecordResponse()
	idStr := fmt.Sprint(id)
	r.Id = &idStr
	r.Name = &name
	r.Type = &recordType
	r.Content = &content
	r.Ttl = &ttl
	return *r
}

func changes() *plan.Changes {
	changes := &plan.Changes{}

	changes.Create = []*endpoint.Endpoint{
		{DNSName: "new.a.de", Targets: endpoint.Targets{"a.de"}, RecordType: "CNAME"},
	}
	changes.Delete = []*endpoint.Endpoint{{DNSName: "b.de", RecordType: "A", Targets: endpoint.Targets{"5.5.5.5"}}}
	changes.UpdateOld = []*endpoint.Endpoint{{DNSName: "a.de", RecordType: "A", Targets: endpoint.Targets{"1.1.1.1", "2.2.2.2"}, RecordTTL: 1000}}
	changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "a.de", RecordType: "A", Targets: endpoint.Targets{"3.3.3.3", "4.4.4.4"}, RecordTTL: 2000}}

	return changes
}

var zoneIdToZoneName = map[string]string{
	"a": "a.de",
	"b": "b.de",
}

var (
	createdRecords = map[string][]sdk.Record{"a": {}, "b": {}}
	deletedRecords = map[string][]string{"a": {}, "b": {}}
)

func isRecordCreated(zoneId string, name string, recordType sdk.RecordTypes, content string, ttl int32) bool {
	for _, record := range createdRecords[zoneId] {
		if *record.Name == name && *record.Type == recordType && *record.Content == content && (ttl == 0 || *record.Ttl == ttl) {
			return true
		}
	}

	return false
}
