package ionoscloud

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionos"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/plan"
	sdk "github.com/ionos-cloud/sdk-go-dns"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func TestNewProvider(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	t.Setenv("IONOS_API_KEY", "1")

	p := NewProvider(endpoint.NewDomainFilter([]string{"a.de."}), &ionos.Configuration{})
	require.Equal(t, true, p.domainFilter.IsConfigured())
	require.Equal(t, false, p.domainFilter.Match("b.de."))

	p = NewProvider(endpoint.DomainFilter{}, &ionos.Configuration{})
	require.Equal(t, false, p.domainFilter.IsConfigured())
	require.Equal(t, true, p.domainFilter.Match("a.de."))
}

func TestRecords(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	testCases := []struct {
		name              string
		givenRecords      sdk.RecordReadList
		givenError        error
		givenDomainFilter endpoint.DomainFilter
		expectedEndpoints []*endpoint.Endpoint
		expectedError     error
	}{
		{
			name:              "no records",
			givenRecords:      sdk.RecordReadList{},
			expectedEndpoints: []*endpoint.Endpoint{},
		},
		{
			name:              "error reading records",
			givenRecords:      sdk.RecordReadList{},
			givenError:        fmt.Errorf("test error"),
			expectedEndpoints: []*endpoint.Endpoint{},
			expectedError:     fmt.Errorf("test error"),
		},
		{
			name: "multiple A records",
			givenRecords: createRecordReadList(3, 0, 0, func(i int) (string, string, string, int32, string) {
				recordName := "a" + fmt.Sprintf("%d", i+1)
				fqdn := recordName + ".a.de"
				return recordName, fqdn, "A", int32((i + 1) * 100), fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)
			}),
			expectedEndpoints: createEndpointSlice(3, func(i int) (string, string, endpoint.TTL, []string) {
				return "a" + fmt.Sprintf("%d", i+1) + ".a.de", "A", endpoint.TTL((i + 1) * 100), []string{fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)}
			}),
		},
		{
			name: "records of Type A and SRV",
			givenRecords: createRecordReadList(2, 0, 333, func(i int) (string, string, string, int32, string) {
				if i == 0 {
					return "a", "a.de", "A", 100, "1.1.1.1"
				}
				return "b", "b.de", "SRV", 200, "server.example.com"
			}),
			expectedEndpoints: createEndpointSlice(2, func(i int) (string, string, endpoint.TTL, []string) {
				if i == 0 {
					return "a.de", "A", 100, []string{"1.1.1.1"}
				}
				return "b.de", "SRV", 200, []string{"333 server.example.com"}
			}),
		},
		{
			name: "multiple records filtered by domain",
			givenRecords: createRecordReadList(6, 0, 0, func(i int) (string, string, string, int32, string) {
				if i < 3 {
					recordName := "a" + fmt.Sprintf("%d", i+1)
					fqdn := recordName + ".a.de"
					return recordName, fqdn, "A", int32((i + 1) * 100), fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)
				}
				recordName := "b" + fmt.Sprintf("%d", i+1)
				fqdn := recordName + ".b.de"
				return recordName, fqdn, "A", int32((i + 1) * 100), fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)
			}),
			givenDomainFilter: endpoint.NewDomainFilter([]string{"a.de"}),
			expectedEndpoints: createEndpointSlice(3, func(i int) (string, string, endpoint.TTL, []string) {
				return "a" + fmt.Sprintf("%d", i+1) + ".a.de", "A", endpoint.TTL((i + 1) * 100), []string{fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)}
			}),
		},
		{
			name: "records mapped to same endpoint",
			givenRecords: createRecordReadList(3, 0, 0, func(i int) (string, string, string, int32, string) {
				if i < 2 {
					return "", "a.de", "A", int32(300), fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)
				} else {
					return "", "c.de", "A", int32(300), fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1)
				}
			}),
			expectedEndpoints: createEndpointSlice(2, func(i int) (string, string, endpoint.TTL, []string) {
				if i == 0 {
					return "a.de", "A", endpoint.TTL(300), []string{"1.1.1.1", "2.2.2.2"}
				} else {
					return "c.de", "A", endpoint.TTL(300), []string{"3.3.3.3"}
				}
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDnsClient := &mockDNSClient{
				allRecords:  tc.givenRecords,
				returnError: tc.givenError,
			}
			provider := &Provider{client: mockDnsClient, domainFilter: tc.givenDomainFilter}
			endpoints, err := provider.Records(ctx)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, endpoints, len(tc.expectedEndpoints))
			assert.ElementsMatch(t, tc.expectedEndpoints, endpoints)
		})
	}
}

func TestApplyChanges(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)
	deZoneId := "deZoneId"
	comZoneId := "comZoneId"
	ctx := context.Background()
	testCases := []struct {
		name                   string
		givenRecords           sdk.RecordReadList
		givenZones             sdk.ZoneReadList
		givenZoneRecords       map[string]sdk.RecordReadList
		givenError             error
		givenDomainFilter      endpoint.DomainFilter
		whenChanges            *plan.Changes
		expectedError          error
		expectedRecordsCreated map[string][]sdk.RecordCreate
		expectedRecordsDeleted map[string][]string
	}{
		{
			name:                   "no changes",
			givenZones:             createZoneReadList(0, nil),
			givenZoneRecords:       map[string]sdk.RecordReadList{},
			whenChanges:            &plan.Changes{},
			expectedRecordsCreated: nil,
			expectedRecordsDeleted: nil,
		},
		{
			name:             "error applying changes",
			givenZones:       createZoneReadList(0, nil),
			givenZoneRecords: map[string]sdk.RecordReadList{},
			givenError:       fmt.Errorf("test error"),
			whenChanges:      &plan.Changes{},
			expectedError:    fmt.Errorf("test error"),
		},
		{
			name: "create one record in a blank zone",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "a.de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(1, func(i int) (string, string, int32, string, int32) {
					return "", "A", int32(300), "1.2.3.4", 0
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "create a SRV record in a blank zone",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "a.de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "SRV", endpoint.TTL(500), []string{"777 myHost.de"}
				}),
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(1, func(i int) (string, string, int32, string, int32) {
					return "", "SRV", int32(500), "myHost.de", 777
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "create a SRV record with no priority field in target",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "a.de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "SRV", endpoint.TTL(700), []string{"myHost.de"}
				}),
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(1, func(i int) (string, string, int32, string, int32) {
					return "", "SRV", int32(700), "myHost.de", 0
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "create a SRV record with wrong priority syntax in target",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "a.de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "SRV", endpoint.TTL(900), []string{"NaN myHost.de"}
				}),
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(1, func(i int) (string, string, int32, string, int32) {
					return "", "SRV", int32(900), "myHost.de", 0
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "create a record which is filtered out from the domain filter",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "a.de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			givenDomainFilter: endpoint.NewDomainFilter([]string{"b.de"}),
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsCreated: nil,
			expectedRecordsDeleted: nil,
		},
		{
			name: "create 2 records from one endpoint in a blank zone",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Create: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4", "5.6.7.8"}
				}),
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(2, func(i int) (string, string, int32, string, int32) {
					if i == 0 {
						return "a", "A", int32(300), "1.2.3.4", 0
					} else {
						return "a", "A", int32(300), "5.6.7.8", 0
					}
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "delete the only record in a zone",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", int32(300), "1.2.3.4"
				}),
			},
			whenChanges: &plan.Changes{
				Delete: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsDeleted: map[string][]string{
				deZoneId: {"0"},
			},
		},
		{
			name: "delete a record which is filtered out from the domain filter",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", int32(300), "1.2.3.4"
				}),
			},
			givenDomainFilter: endpoint.NewDomainFilter([]string{"b.de"}),
			whenChanges: &plan.Changes{
				Delete: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "delete multiple records, in different zones",
			givenZones: createZoneReadList(2, func(i int) (string, string) {
				if i == 0 {
					return deZoneId, "de"
				} else {
					return comZoneId, "com"
				}
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(2, 0, 0, func(n int) (string, string, string, int32, string) {
					if n == 0 {
						return "a", "a.de", "A", 300, "1.2.3.4"
					} else {
						return "a", "a.de", "A", 300, "5.6.7.8"
					}
				}),
				comZoneId: createRecordReadList(1, 2, 0, func(n int) (string, string, string, int32, string) {
					return "a", "a.com", "A", 300, "11.22.33.44"
				}),
			},
			whenChanges: &plan.Changes{
				Delete: createEndpointSlice(2, func(i int) (string, string, endpoint.TTL, []string) {
					if i == 0 {
						return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4", "5.6.7.8"}
					} else {
						return "a.com", "A", endpoint.TTL(300), []string{"11.22.33.44"}
					}
				}),
			},
			expectedRecordsDeleted: map[string][]string{
				deZoneId:  {"0", "1"},
				comZoneId: {"2"},
			},
		},
		{
			name: "delete record which is not in the zone, deletes nothing",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(0, 0, 0, nil),
			},
			whenChanges: &plan.Changes{
				Delete: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsDeleted: nil,
		},
		{
			name: "delete one record from targets part of endpoint",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", 300, "1.2.3.4"
				}),
			},
			whenChanges: &plan.Changes{
				Delete: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4", "5.6.7.8"}
				}),
			},
			expectedRecordsDeleted: map[string][]string{
				deZoneId: {"0"},
			},
		},
		{
			name: "update single record",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", 300, "1.2.3.4"
				}),
			},
			whenChanges: &plan.Changes{
				UpdateOld: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
				UpdateNew: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"5.6.7.8"}
				}),
			},
			expectedRecordsDeleted: map[string][]string{
				deZoneId: {"0"},
			},
			expectedRecordsCreated: map[string][]sdk.RecordCreate{
				deZoneId: createRecordCreateSlice(1, func(i int) (string, string, int32, string, int32) {
					return "a", "A", 300, "5.6.7.8", 0
				}),
			},
		},
		{
			name: "update a record which is filtered out by domain filter, does nothing",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", 300, "1.2.3.4"
				}),
			},
			givenDomainFilter: endpoint.NewDomainFilter([]string{"b.de"}),

			whenChanges: &plan.Changes{
				UpdateOld: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
				UpdateNew: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"5.6.7.8"}
				}),
			},
			expectedRecordsDeleted: nil,
			expectedRecordsCreated: nil,
		},
		{
			name: "update when old and new endpoint are the same, does nothing",
			givenZones: createZoneReadList(1, func(i int) (string, string) {
				return deZoneId, "de"
			}),
			givenZoneRecords: map[string]sdk.RecordReadList{
				deZoneId: createRecordReadList(1, 0, 0, func(i int) (string, string, string, int32, string) {
					return "a", "a.de", "A", 300, "1.2.3.4"
				}),
			},
			whenChanges: &plan.Changes{
				UpdateOld: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
				UpdateNew: createEndpointSlice(1, func(i int) (string, string, endpoint.TTL, []string) {
					return "a.de", "A", endpoint.TTL(300), []string{"1.2.3.4"}
				}),
			},
			expectedRecordsDeleted: nil,
			expectedRecordsCreated: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDnsClient := &mockDNSClient{
				allRecords:  tc.givenRecords,
				allZones:    tc.givenZones,
				zoneRecords: tc.givenZoneRecords,
				returnError: tc.givenError,
			}
			provider := &Provider{client: mockDnsClient, domainFilter: tc.givenDomainFilter}
			err := provider.ApplyChanges(ctx, tc.whenChanges)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, mockDnsClient.createdRecords, len(tc.expectedRecordsCreated))
			for zoneId, expectedRecordsCreated := range tc.expectedRecordsCreated {
				actualRecords, ok := mockDnsClient.createdRecords[zoneId]
				require.True(t, ok)
				for i, actualRecord := range actualRecords {
					expJson, _ := expectedRecordsCreated[i].MarshalJSON()
					actJson, _ := actualRecord.MarshalJSON()
					require.Equal(t, expJson, actJson)
				}
			}
			for zoneId, expectedDeletedRecordIds := range tc.expectedRecordsDeleted {
				require.Len(t, mockDnsClient.deletedRecords[zoneId], len(expectedDeletedRecordIds), "deleted records in zone '%s' do not fit", zoneId)
				actualDeletedRecordIds, ok := mockDnsClient.deletedRecords[zoneId]
				require.True(t, ok)
				assert.ElementsMatch(t, expectedDeletedRecordIds, actualDeletedRecordIds)
			}
		})
	}
}

func TestPropertyValuesEqual(t *testing.T) {
	provider := &Provider{}
	name := RandStringRunes(10)
	require.True(t, provider.PropertyValuesEqual(name, "a", "a"))
	require.False(t, provider.PropertyValuesEqual(name, "a", "b"))
}

func TestAdjustEndpoints(t *testing.T) {
	provider := &Provider{}
	endpoints := createEndpointSlice(rand.Intn(5), func(i int) (string, string, endpoint.TTL, []string) {
		return RandStringRunes(10), RandStringRunes(1), endpoint.TTL(300), []string{RandStringRunes(5)}
	})
	actualEndpoints := provider.AdjustEndpoints(endpoints)
	require.Equal(t, endpoints, actualEndpoints)
}

func TestReadMaxRecords(t *testing.T) {
	provider := &Provider{client: pagingMockDNSService{t: t}}
	endpoints, err := provider.Records(context.Background())
	require.NoError(t, err)
	require.Len(t, endpoints, recordReadMaxCount)
}

func TestReadMaxZones(t *testing.T) {
	provider := &Provider{client: pagingMockDNSService{t: t}}
	zt, err := provider.createZoneTree(context.Background())
	require.NoError(t, err)
	require.Equal(t, zoneReadMaxCount, zt.GetZonesCount())
}

type pagingMockDNSService struct {
	t *testing.T
}

func (p pagingMockDNSService) GetAllRecords(ctx context.Context, offset int32) (sdk.RecordReadList, error) {
	require.Equal(p.t, 0, int(offset)%recordReadLimit)
	records := createRecordReadList(recordReadLimit, int(offset), 0, func(i int) (string, string, string, int32, string) {
		recordName := fmt.Sprintf("a%d", int(offset)+i)
		return recordName, recordName + ".de", "A", 300, "1.1.1.1"
	})
	return records, nil
}

func (pagingMockDNSService) GetZoneRecords(ctx context.Context, zoneId string) (sdk.RecordReadList, error) {
	panic("implement me")
}

func (pagingMockDNSService) GetRecordsByZoneIdAndName(ctx context.Context, zoneId, name string) (sdk.RecordReadList, error) {
	panic("implement me")
}

func (p pagingMockDNSService) GetZones(ctx context.Context, offset int32) (sdk.ZoneReadList, error) {
	require.Equal(p.t, 0, int(offset)%zoneReadLimit)
	zones := createZoneReadList(zoneReadLimit, func(i int) (string, string) {
		idStr := fmt.Sprintf("%d", int(offset)+i)
		return idStr, fmt.Sprintf("zone%s.de", idStr)
	})
	return zones, nil
}

func (pagingMockDNSService) GetZone(ctx context.Context, zoneId string) (sdk.ZoneRead, error) {
	panic("implement me")
}

func (pagingMockDNSService) DeleteRecord(ctx context.Context, zoneId string, recordId string) error {
	panic("implement me")
}

func (pagingMockDNSService) CreateRecord(ctx context.Context, zoneId string, record sdk.RecordCreate) error {
	panic("implement me")
}

type mockDNSClient struct {
	returnError    error
	allRecords     sdk.RecordReadList
	zoneRecords    map[string]sdk.RecordReadList
	allZones       sdk.ZoneReadList
	createdRecords map[string][]sdk.RecordCreate // zoneId -> recordCreates
	deletedRecords map[string][]string           // zoneId -> recordIds
}

func (c *mockDNSClient) GetAllRecords(ctx context.Context, offset int32) (sdk.RecordReadList, error) {
	log.Debugf("GetAllRecords called")
	return c.allRecords, c.returnError
}

func (c *mockDNSClient) GetZoneRecords(ctx context.Context, zoneId string) (sdk.RecordReadList, error) {
	log.Debugf("GetZoneRecords called with zoneId %s", zoneId)
	return c.zoneRecords[zoneId], c.returnError
}

func (c *mockDNSClient) GetRecordsByZoneIdAndName(ctx context.Context, zoneId, name string) (sdk.RecordReadList, error) {
	log.Debugf("GetRecordsByZoneIdAndName called with zoneId %s and name %s", zoneId, name)
	result := make([]sdk.RecordRead, 0)
	recordsOfZone := c.zoneRecords[zoneId]
	for _, recordRead := range *recordsOfZone.GetItems() {
		if *recordRead.GetProperties().GetName() == name {
			result = append(result, recordRead)
		}
	}
	return sdk.RecordReadList{Items: &result}, c.returnError
}

func (c *mockDNSClient) GetZones(ctx context.Context, offset int32) (sdk.ZoneReadList, error) {
	log.Debug("GetZones called ")
	if c.allZones.HasItems() {
		for _, zone := range *c.allZones.GetItems() {
			log.Debugf("GetZones: zone '%s' with id '%s'", *zone.GetProperties().GetZoneName(), *zone.GetId())
		}
	} else {
		log.Debug("GetZones: no zones")
	}
	return c.allZones, c.returnError
}

func (c *mockDNSClient) GetZone(ctx context.Context, zoneId string) (sdk.ZoneRead, error) {
	log.Debugf("GetZone called with zoneId '%s", zoneId)
	for _, zone := range *c.allZones.GetItems() {
		if *zone.GetId() == zoneId {
			return zone, nil
		}
	}
	return *sdk.NewZoneReadWithDefaults(), c.returnError
}

func (c *mockDNSClient) CreateRecord(ctx context.Context, zoneId string, record sdk.RecordCreate) error {
	log.Debugf("CreateRecord called with zoneId %s and record %v", zoneId, record)
	if c.createdRecords == nil {
		c.createdRecords = make(map[string][]sdk.RecordCreate)
	}
	c.createdRecords[zoneId] = append(c.createdRecords[zoneId], record)
	return c.returnError
}

func (c *mockDNSClient) DeleteRecord(ctx context.Context, zoneId string, recordId string) error {
	log.Debugf("DeleteRecord called with zoneId %s and recordId %s", zoneId, recordId)
	if c.deletedRecords == nil {
		c.deletedRecords = make(map[string][]string)
	}
	c.deletedRecords[zoneId] = append(c.deletedRecords[zoneId], recordId)
	return c.returnError
}

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createZoneReadList(count int, modifier func(int) (string, string)) sdk.ZoneReadList {
	zones := make([]sdk.ZoneRead, count)
	for i := 0; i < count; i++ {
		id, name := modifier(i)
		zones[i] = sdk.ZoneRead{
			Id: sdk.PtrString(id),
			Properties: &sdk.Zone{
				ZoneName: sdk.PtrString(name),
				Enabled:  sdk.PtrBool(true),
			},
		}
	}
	return sdk.ZoneReadList{Items: &zones}
}

func createRecordCreateSlice(count int, modifier func(int) (string, string, int32, string, int32)) []sdk.RecordCreate {
	records := make([]sdk.RecordCreate, count)
	for i := 0; i < count; i++ {
		name, typ, ttl, content, prio := modifier(i)
		records[i] = sdk.RecordCreate{
			Properties: &sdk.Record{
				Name:    sdk.PtrString(name),
				Type:    sdk.PtrString(typ),
				Ttl:     sdk.PtrInt32(ttl),
				Content: sdk.PtrString(content),
				Enabled: sdk.PtrBool(true),
			},
		}
		if prio != 0 {
			records[i].Properties.SetPriority(prio)
		}
	}
	return records
}

func createRecordReadList(count, idOffset int, priority int32, modifier func(int) (string, string, string, int32, string)) sdk.RecordReadList {
	records := make([]sdk.RecordRead, count)
	for i := 0; i < count; i++ {
		name, fqdn, typ, ttl, content := modifier(i)
		// use random number as id
		id := i + idOffset
		records[i] = sdk.RecordRead{
			Id: sdk.PtrString(fmt.Sprintf("%d", id)),
			Properties: &sdk.Record{
				Name:     sdk.PtrString(name),
				Type:     sdk.PtrString(typ),
				Ttl:      sdk.PtrInt32(ttl),
				Content:  sdk.PtrString(content),
				Priority: sdk.PtrInt32(priority),
			},
			Metadata: &sdk.MetadataWithStateFqdnZoneId{
				Fqdn: sdk.PtrString(fqdn),
			},
		}
	}
	return sdk.RecordReadList{Items: &records}
}

func createEndpointSlice(count int, modifier func(int) (string, string, endpoint.TTL, []string)) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, count)
	for i := 0; i < count; i++ {
		name, typ, ttl, targets := modifier(i)
		endpoints[i] = &endpoint.Endpoint{
			DNSName:    name,
			RecordType: typ,
			Targets:    targets,
			RecordTTL:  ttl,
			Labels:     map[string]string{},
		}
	}
	return endpoints
}
