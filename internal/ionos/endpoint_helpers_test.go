package ionos

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/stretchr/testify/require"
)

type myRecord struct {
	name    string
	content string
}

func TestRetrieveRecords(t *testing.T) {
	myRecords := []myRecord{
		{"a.com", "content1-a.com"},
		{"a.com", "content2-a.com"},
		{"b.com", "content-b.com"},
	}
	eps := NewEndpointCollection[myRecord](myRecords,
		func(record myRecord) *endpoint.Endpoint {
			return &endpoint.Endpoint{
				DNSName:    record.name,
				RecordType: "A",
				Targets:    []string{record.content},
				RecordTTL:  300,
			}
		}, func(record myRecord) string {
			return record.name
		})
	endPoints := eps.RetrieveEndPoints()
	require.EqualValues(t, 2, len(endPoints))
	sort.Slice(endPoints, func(i, j int) bool {
		return endPoints[i].DNSName < endPoints[j].DNSName
	})
	require.EqualValues(t, endPoints[0].DNSName, "a.com")
	require.EqualValues(t, endPoints[0].RecordType, "A")
	require.EqualValues(t, endPoints[0].RecordTTL, 300)
	assert.Contains(t, endPoints[0].Targets, "content1-a.com")
	assert.Contains(t, endPoints[0].Targets, "content2-a.com")

	require.EqualValues(t, endPoints[1].DNSName, "b.com")
	require.EqualValues(t, endPoints[1].RecordType, "A")
	require.EqualValues(t, endPoints[1].RecordTTL, 300)
	assert.Contains(t, endPoints[1].Targets, "content-b.com")
}

type myZone struct {
	name string
}

func TestFindZoneByName(t *testing.T) {
	myZones := []*myZone{
		{"a.com"},
		{"a1.a.com"},
		{"a2.a.com"},
		{"b.com"},
		{"org"},
	}
	zt := NewZoneTree[*myZone]()

	for _, zone := range myZones {
		zt.AddZone(zone, zone.name)
	}

	require.EqualValues(t, myZones[0], zt.FindZoneByDomainName("a.com"))

	require.EqualValues(t, myZones[1], zt.FindZoneByDomainName("a1.a.com"))

	require.EqualValues(t, myZones[2], zt.FindZoneByDomainName("a2.a.com"))

	require.EqualValues(t, myZones[0], zt.FindZoneByDomainName("b.a.com"))

	require.EqualValues(t, myZones[0], zt.FindZoneByDomainName("e.f.goo.a.com"))

	require.EqualValues(t, myZones[3], zt.FindZoneByDomainName("b.com"))

	require.EqualValues(t, myZones[3], zt.FindZoneByDomainName("a.b.com"))
	require.EqualValues(t, myZones[0], zt.FindZoneByDomainName("a.a.com"))

	require.EqualValues(t, myZones[3], zt.FindZoneByDomainName("e.b.com"))

	require.EqualValues(t, myZones[4], zt.FindZoneByDomainName("org"))
	require.EqualValues(t, myZones[4], zt.FindZoneByDomainName("de.org"))
	require.EqualValues(t, myZones[4], zt.FindZoneByDomainName("org.org"))

	require.Nil(t, zt.FindZoneByDomainName("com"))
	require.Nil(t, zt.FindZoneByDomainName("com.a"))
}
