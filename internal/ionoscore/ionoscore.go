package ionoscore

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionos"

	log "github.com/sirupsen/logrus"

	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/plan"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/provider"
	sdk "github.com/ionos-developer/dns-sdk-go"
)

// Provider implements the DNS provider for IONOS DNS.
type Provider struct {
	provider.BaseProvider
	client       DnsService
	domainFilter endpoint.DomainFilter
	dryRun       bool
}

// DnsService interface to the dns backend, also needed for creating mocks in tests
type DnsService interface {
	GetZones(ctx context.Context) ([]sdk.Zone, error)
	GetZone(ctx context.Context, zoneId string) (*sdk.CustomerZone, error)
	CreateRecords(ctx context.Context, zoneId string, records []sdk.Record) error
	DeleteRecord(ctx context.Context, zoneId string, recordId string) error
}

// DnsClient client of the dns api
type DnsClient struct {
	client *sdk.APIClient
}

// GetZones client get zones method
func (c DnsClient) GetZones(ctx context.Context) ([]sdk.Zone, error) {
	zones, _, err := c.client.ZonesApi.GetZones(ctx).Execute()
	if err != nil {
		return nil, err
	}

	return zones, err
}

// GetZone client get zone method
func (c DnsClient) GetZone(ctx context.Context, zoneId string) (*sdk.CustomerZone, error) {
	zoneInfo, _, err := c.client.ZonesApi.GetZone(ctx, zoneId).Execute()
	return zoneInfo, err
}

// CreateRecords client create records method
func (c DnsClient) CreateRecords(ctx context.Context, zoneId string, records []sdk.Record) error {
	_, _, err := c.client.RecordsApi.CreateRecords(ctx, zoneId).Record(records).Execute()
	return err
}

// DeleteRecord client delete record method
func (c DnsClient) DeleteRecord(ctx context.Context, zoneId string, recordId string) error {
	_, err := c.client.RecordsApi.DeleteRecord(ctx, zoneId, recordId).Execute()
	return err
}

// NewProvider creates a new IONOS DNS provider.
func NewProvider(domainFilter endpoint.DomainFilter, configuration *ionos.Configuration, dryRun bool) (*Provider, error) {
	client, err := createClient(configuration)
	if err != nil {
		return nil, fmt.Errorf("provider creation failed, %v", err)
	}

	prov := &Provider{
		client:       DnsClient{client: client},
		domainFilter: domainFilter,
		dryRun:       dryRun,
	}

	return prov, nil
}

func createClient(config *ionos.Configuration) (*sdk.APIClient, error) {
	maskAPIKey := func() string {
		if len(config.APIKey) <= 3 {
			return strings.Repeat("*", len(config.APIKey))
		}
		return fmt.Sprintf("%s%s", config.APIKey[:3], strings.Repeat("*", len(config.APIKey)-3))
	}
	log.Infof(
		"Creating ionos core DNS client with parameters: API Endpoint URL: '%v', Auth header: '%v', API key: '%v', Debug: '%v'",
		config.APIEndpointURL,
		config.AuthHeader,
		maskAPIKey(),
		config.Debug,
	)

	sdkConfig := sdk.NewConfiguration()
	if config.APIEndpointURL != "" {
		sdkConfig.Servers[0].URL = config.APIEndpointURL
	}
	sdkConfig.AddDefaultHeader(config.AuthHeader, config.APIKey)
	sdkConfig.UserAgent = fmt.Sprintf(
		"external-dns os %s arch %s",
		runtime.GOOS, runtime.GOARCH)
	sdkConfig.Debug = config.Debug

	return sdk.NewAPIClient(sdkConfig), nil
}

// Records returns the list of resource records in all zones.
func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.getZones(ctx)
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint

	for zoneId := range zones {
		zoneInfo, err := p.client.GetZone(ctx, zoneId)
		if err != nil {
			log.Warnf("Failed to fetch zoneId %v: %v", zoneId, err)
			continue
		}

		recordSets := map[string]*endpoint.Endpoint{}
		for _, r := range zoneInfo.Records {
			key := *r.Name + "/" + getType(r) + "/" + strconv.Itoa(int(*r.Ttl))
			if rrset, ok := recordSets[key]; ok {
				rrset.Targets = append(rrset.Targets, *r.Content)
			} else {
				recordSets[key] = recordToEndpoint(r)
			}
		}

		for _, ep := range recordSets {
			endpoints = append(endpoints, ep)
		}
	}
	log.Debugf("Records() found %d endpoints: %v", len(endpoints), endpoints)
	return endpoints, nil
}

// ApplyChanges applies a given set of changes.
func (p *Provider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	zones, err := p.getZones(ctx)
	if err != nil {
		return err
	}

	toCreate := make([]*endpoint.Endpoint, len(changes.Create))
	copy(toCreate, changes.Create)

	toDelete := make([]*endpoint.Endpoint, len(changes.Delete))
	copy(toDelete, changes.Delete)

	for i, updateOldEndpoint := range changes.UpdateOld {
		if !sameEndpoints(*updateOldEndpoint, *changes.UpdateNew[i]) {
			toDelete = append(toDelete, updateOldEndpoint)
			toCreate = append(toCreate, changes.UpdateNew[i])
		}
	}

	zonesToDeleteFrom := p.fetchZonesToDeleteFrom(ctx, toDelete, zones)

	for _, e := range toDelete {
		zoneId := getHostZoneID(e.DNSName, zones)
		if zoneId == "" {
			log.Warnf("No zone to delete %v from", e)
			continue
		}

		if zone, ok := zonesToDeleteFrom[zoneId]; ok {
			p.deleteEndpoint(ctx, e, zone)
		} else {
			log.Warnf("No zone to delete %v from", e)
		}
	}

	for _, e := range toCreate {
		p.createEndpoint(ctx, e, zones)
	}

	return nil
}

// fetchZonesToDeleteFrom fetches all the zones that will be performed deletions upon.
func (p *Provider) fetchZonesToDeleteFrom(ctx context.Context, toDelete []*endpoint.Endpoint, zones map[string]string) map[string]*sdk.CustomerZone {
	zonesIdsToDeleteFrom := map[string]bool{}
	for _, e := range toDelete {
		zoneId := getHostZoneID(e.DNSName, zones)
		if zoneId != "" {
			zonesIdsToDeleteFrom[zoneId] = true
		}
	}

	zonesToDeleteFrom := map[string]*sdk.CustomerZone{}
	for zoneId := range zonesIdsToDeleteFrom {
		zone, err := p.client.GetZone(ctx, zoneId)
		if err == nil {
			zonesToDeleteFrom[zoneId] = zone
		}
	}

	return zonesToDeleteFrom
}

// deleteEndpoint deletes all resource records for the endpoint through the IONOS DNS API.
func (p *Provider) deleteEndpoint(ctx context.Context, e *endpoint.Endpoint, zone *sdk.CustomerZone) {
	log.Infof("Delete endpoint %v", e)
	if p.dryRun {
		return
	}

	for _, target := range e.Targets {
		recordId := ""
		for _, record := range zone.Records {
			if *record.Name == e.DNSName && getType(record) == e.RecordType && *record.Content == target {
				recordId = *record.Id
				break
			}
		}

		if recordId == "" {
			log.Warnf("Record %v %v %v not found in zone", e.DNSName, e.RecordType, target)
			continue
		}

		if p.client.DeleteRecord(ctx, *zone.Id, recordId) != nil {
			log.Warnf("Failed to delete record %v %v %v", e.DNSName, e.RecordType, target)
		}
	}
}

// createEndpoint creates the record set for the endpoint using the IONOS DNS API.
func (p *Provider) createEndpoint(ctx context.Context, e *endpoint.Endpoint, zones map[string]string) {
	log.Infof("Create endpoint %v", e)
	if p.dryRun {
		return
	}

	zoneId := getHostZoneID(e.DNSName, zones)
	if zoneId == "" {
		log.Warnf("No zone to create %v into", e)
		return
	}

	records := endpointToRecords(e)
	if p.client.CreateRecords(ctx, zoneId, records) != nil {
		log.Warnf("Failed to create record for %v", e)
	}
}

// endpointToRecords converts an endpoint to a slice of records.
func endpointToRecords(endpoint *endpoint.Endpoint) []sdk.Record {
	records := make([]sdk.Record, 0)

	for _, target := range endpoint.Targets {
		record := sdk.NewRecord()

		record.SetName(endpoint.DNSName)
		record.SetType(sdk.RecordTypes(endpoint.RecordType))
		record.SetContent(target)

		ttl := int32(endpoint.RecordTTL)
		if ttl != 0 {
			record.SetTtl(ttl)
		}

		records = append(records, *record)
	}

	return records
}

// recordToEndpoint converts a record to an endpoint.
func recordToEndpoint(r sdk.RecordResponse) *endpoint.Endpoint {
	return endpoint.NewEndpointWithTTL(*r.Name, getType(r), endpoint.TTL(*r.Ttl), *r.Content)
}

// getZones returns a ZoneID -> ZoneName mapping for zones that match domain filter.
func (p *Provider) getZones(ctx context.Context) (map[string]string, error) {
	zones, err := p.client.GetZones(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}

	for _, zone := range zones {
		if p.domainFilter.Match(*zone.Name) {
			result[*zone.Id] = *zone.Name
		}
	}

	return result, nil
}

// getHostZoneID finds the best suitable DNS zone for the hostname.
func getHostZoneID(hostname string, zones map[string]string) string {
	longestZoneLength := 0
	resultID := ""

	for zoneID, zoneName := range zones {
		if !strings.HasSuffix(hostname, zoneName) {
			continue
		}
		ln := len(zoneName)
		if ln > longestZoneLength {
			resultID = zoneID
			longestZoneLength = ln
		}
	}

	return resultID
}

// getType returns the record type as string.
func getType(record sdk.RecordResponse) string {
	return string(*record.Type)
}

// sameEndpoints returns if the two endpoints have the same values.
func sameEndpoints(a endpoint.Endpoint, b endpoint.Endpoint) bool {
	return a.DNSName == b.DNSName && a.RecordType == b.RecordType && a.RecordTTL == b.RecordTTL && a.Targets.Same(b.Targets)
}
