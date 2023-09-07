package ionoscloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionos"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/plan"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/provider"
	sdk "github.com/ionos-cloud/sdk-go-dns"
	log "github.com/sirupsen/logrus"
)

const (
	logFieldZoneID        = "zoneID"
	logFieldRecordID      = "recordID"
	logFieldRecordName    = "recordName"
	logFieldRecordFQDN    = "recordFQDN"
	logFieldRecordType    = "recordType"
	logFieldRecordContent = "recordContent"
	logFieldRecordTTL     = "recordTTL"
	logFieldDomainFilter  = "domainFilter"
	// max number of records to read per request
	recordReadLimit = 1000
	// max number of records to read in total
	recordReadMaxCount = 10 * recordReadLimit
	// max number of zones to read per request
	zoneReadLimit = 1000
	// max number of zones to read in total
	zoneReadMaxCount = 10 * zoneReadLimit

	recordTypeSRV = "SRV"
)

type DNSClient struct {
	client *sdk.APIClient
	dryRun bool
}

type DNSService interface {
	GetAllRecords(ctx context.Context, offset int32) (sdk.RecordReadList, error)
	GetRecordsByZoneIdAndName(ctx context.Context, zoneId, name string) (sdk.RecordReadList, error)
	GetZones(ctx context.Context, offset int32) (sdk.ZoneReadList, error)
	DeleteRecord(ctx context.Context, zoneId string, recordId string) error
	CreateRecord(ctx context.Context, zoneId string, record sdk.RecordCreate) error
}

// GetAllRecords retrieve all records https://github.com/ionos-cloud/sdk-go-dns/blob/master/docs/api/RecordsApi.md#recordsget
func (c *DNSClient) GetAllRecords(ctx context.Context, offset int32) (sdk.RecordReadList, error) {
	log.Debugf("get all records with offset %d ...", offset)
	records, _, err := c.client.RecordsApi.RecordsGet(ctx).Limit(recordReadLimit).Offset(offset).FilterState(sdk.AVAILABLE).Execute()
	if err != nil {
		log.Errorf("failed to get all records: %v", err)
		return records, err
	}
	if records.HasItems() {
		log.Debugf("found %d records", len(*records.Items))
	} else {
		log.Debug("no records found")
	}
	return records, err
}

func (c *DNSClient) GetRecordsByZoneIdAndName(ctx context.Context, zoneId, name string) (sdk.RecordReadList, error) {
	logger := log.WithField(logFieldZoneID, zoneId).WithField(logFieldRecordName, name)
	logger.Debug("get records from zone by name ...")
	records, _, err := c.client.RecordsApi.RecordsGet(ctx).FilterZoneId(zoneId).FilterName(name).
		FilterState(sdk.AVAILABLE).Execute()
	if err != nil {
		logger.Errorf("failed to get records from zone by name: %v", err)
		return records, err
	}
	if records.HasItems() {
		logger.Debugf("found %d records", len(*records.Items))
	} else {
		logger.Debug("no records found")
	}
	return records, nil
}

// GetZones client get zones method
func (c *DNSClient) GetZones(ctx context.Context, offset int32) (sdk.ZoneReadList, error) {
	log.Debug("get all zones ...")
	zones, _, err := c.client.ZonesApi.ZonesGet(ctx).Offset(offset).Limit(zoneReadLimit).FilterState(sdk.AVAILABLE).Execute()
	if err != nil {
		log.Errorf("failed to get all zones: %v", err)
		return zones, err
	}
	if zones.HasItems() {
		log.Debugf("found %d zones", len(*zones.Items))
	} else {
		log.Debug("no zones found")
	}
	return zones, err
}

// CreateRecord client create record method
func (c *DNSClient) CreateRecord(ctx context.Context, zoneId string, record sdk.RecordCreate) error {
	recordProps := record.GetProperties()
	logger := log.WithField(logFieldZoneID, zoneId).WithField(logFieldRecordName, *recordProps.GetName()).
		WithField(logFieldRecordType, *recordProps.GetType()).WithField(logFieldRecordContent, *recordProps.GetContent()).
		WithField(logFieldRecordTTL, *recordProps.GetTtl())
	logger.Debugf("creating record ...")
	if !c.dryRun {
		recordRead, _, err := c.client.RecordsApi.ZonesRecordsPost(ctx, zoneId).RecordCreate(record).Execute()
		if err != nil {
			logger.Errorf("failed to create record: %v", err)
			return err
		}
		logger.Debugf("created successfully record with id: '%s'", *recordRead.GetId())
	} else {
		logger.Info("** DRY RUN **, record not created")
	}
	return nil
}

// DeleteRecord client delete record method
func (c *DNSClient) DeleteRecord(ctx context.Context, zoneId string, recordId string) error {
	logger := log.WithField(logFieldZoneID, zoneId).WithField(logFieldRecordID, recordId)
	logger.Debugf("deleting record: %v ...", recordId)
	if !c.dryRun {
		_, err := c.client.RecordsApi.ZonesRecordsDelete(ctx, zoneId, recordId).Execute()
		if err != nil {
			logger.Errorf("failed to delete record: %v", err)
			return err
		}
		logger.Debug("record deleted successfully")
	} else {
		logger.Info("** DRY RUN **, record not deleted")
	}
	return nil
}

// Provider extends base provider to work with paas dns rest API
type Provider struct {
	provider.BaseProvider
	client DNSService
}

// NewProvider returns an instance of new provider
func NewProvider(baseProvider *provider.BaseProvider, configuration *ionos.Configuration) *Provider {
	client := createClient(configuration)
	prov := &Provider{
		BaseProvider: *baseProvider,
		client:       &DNSClient{client: client, dryRun: configuration.DryRun},
	}
	return prov
}

func createClient(ionosConfig *ionos.Configuration) *sdk.APIClient {
	jwtString := func() string {
		split := strings.Split(ionosConfig.APIKey, ".")
		if len(split) == 3 {
			headerBytes, _ := base64.RawStdEncoding.DecodeString(split[0])
			payloadBytes, _ := base64.RawStdEncoding.DecodeString(split[1])
			return fmt.Sprintf("JWT-header: %s, JWT-payload: %s", headerBytes, payloadBytes)
		}
		return ""
	}
	log.Infof(
		"Creating ionos cloud DNS client with parameters: API Endpoint URL: '%v', Auth header: '%v', Debug: '%v'",
		ionosConfig.APIEndpointURL,
		ionosConfig.AuthHeader,
		ionosConfig.Debug,
	)
	log.Debugf("JWT: %s", jwtString())

	if ionosConfig.DryRun {
		log.Warnf("*** Dry run is enabled, no changes will be made to ionos cloud DNS ***")
	}

	sdkConfig := sdk.NewConfiguration("", "", ionosConfig.APIKey, ionosConfig.APIEndpointURL)
	sdkConfig.Debug = ionosConfig.Debug
	apiClient := sdk.NewAPIClient(sdkConfig)
	return apiClient
}

func (p *Provider) readAllRecords(ctx context.Context) ([]sdk.RecordRead, error) {
	var result []sdk.RecordRead
	offset := int32(0)
	for {
		recordReadList, err := p.client.GetAllRecords(ctx, offset)
		if err != nil {
			return nil, err
		}
		if recordReadList.HasItems() {
			items := *recordReadList.GetItems()
			result = append(result, items...)
			offset += recordReadLimit
			if len(items) < recordReadLimit || offset >= recordReadMaxCount {
				break
			}
		} else {
			break
		}
	}

	if p.BaseProvider.GetDomainFilter().IsConfigured() {
		filteredResult := make([]sdk.RecordRead, 0)
		for _, record := range result {
			fqdn := *record.GetMetadata().GetFqdn()
			if p.BaseProvider.GetDomainFilter().Match(fqdn) {
				filteredResult = append(filteredResult, record)
			}
		}
		logger := log.WithField(logFieldDomainFilter, p.BaseProvider.GetDomainFilter())
		logger.Debugf("found %d records after applying domainFilter", len(filteredResult))
		return filteredResult, nil
	} else {
		return result, nil
	}
}

func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	allRecords, err := p.readAllRecords(ctx)
	if err != nil {
		return nil, err
	}
	epCollection := ionos.NewEndpointCollection[sdk.RecordRead](allRecords,
		func(recordRead sdk.RecordRead) *endpoint.Endpoint {
			recordProperties := *recordRead.GetProperties()
			recordMetadata := *recordRead.GetMetadata()
			target := *recordProperties.GetContent()
			priority, hasPriority := recordProperties.GetPriorityOk()
			if *recordProperties.GetType() == recordTypeSRV && hasPriority {
				target = fmt.Sprintf("%d %s", *priority, target)
			}
			return endpoint.NewEndpointWithTTL(*recordMetadata.GetFqdn(), *recordProperties.GetType(),
				endpoint.TTL(*recordProperties.GetTtl()), target)
		}, func(recordRead sdk.RecordRead) string {
			recordProperties := *recordRead.GetProperties()
			recordMetadata := *recordRead.GetMetadata()
			return *recordMetadata.GetFqdn() + "/" + *recordProperties.GetType() + "/" + strconv.Itoa(int(*recordProperties.GetTtl()))
		})
	return epCollection.RetrieveEndPoints(), nil
}

func (p *Provider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	epToCreate, epToDelete := ionos.GetCreateDeleteSetsFromChanges(changes)
	zt, err := p.createZoneTree(ctx)
	if err != nil {
		return err
	}
	recordsToDelete := ionos.NewRecordCollection[sdk.RecordRead](epToDelete, func(ep *endpoint.Endpoint) []sdk.RecordRead {
		logger := log.WithField(logFieldRecordFQDN, ep.DNSName)
		records := make([]sdk.RecordRead, 0)
		zone := zt.FindZoneByDomainName(ep.DNSName)
		if zone.Id == nil {
			logger.Error("no zone found for record")
			return records
		}
		logger = logger.WithField(logFieldZoneID, *zone.GetId())
		recordName := extractRecordName(ep.DNSName, zone)
		zoneRecordReadList, err := p.client.GetRecordsByZoneIdAndName(ctx, *zone.GetId(), recordName)
		if err != nil {
			logger.Errorf("failed to get records for zone, error: %v", err)
			return records
		}
		if !zoneRecordReadList.HasItems() {
			logger.Warn("no records found to delete for zone")
			return records
		}
		result := make([]sdk.RecordRead, 0)
		for _, recordRead := range *zoneRecordReadList.GetItems() {
			record := *recordRead.GetProperties()
			if *record.GetType() == ep.RecordType {
				for _, target := range ep.Targets {
					if *record.GetContent() == target {
						result = append(result, recordRead)
					}
				}
			}
		}
		if len(result) == 0 {
			logger.Warnf("no records in zone fit to delete for endpoint: %v", ep)
		}
		return result
	})

	if err := recordsToDelete.ForEach(func(ep *endpoint.Endpoint, recordRead sdk.RecordRead) error {
		domainName := *recordRead.GetMetadata().GetFqdn()
		zone := zt.FindZoneByDomainName(domainName)
		if !zone.HasId() {
			return fmt.Errorf("no zone found for domain '%s'", domainName)
		}
		err := p.client.DeleteRecord(ctx, *zone.GetId(), *recordRead.GetId())
		return err
	}); err != nil {
		return err
	}

	recordsToCreate := ionos.NewRecordCollection[*sdk.RecordCreate](epToCreate, func(ep *endpoint.Endpoint) []*sdk.RecordCreate {
		logger := log.WithField(logFieldRecordFQDN, ep.DNSName).WithField(logFieldRecordType, ep.RecordType)
		zone := zt.FindZoneByDomainName(ep.DNSName)
		if !zone.HasId() {
			logger.Warnf("no zone found for domain '%s', skipping record creation", ep.DNSName)
			return nil
		}
		recordName := extractRecordName(ep.DNSName, zone)
		result := make([]*sdk.RecordCreate, 0)
		for _, target := range ep.Targets {
			content := target
			priority := int32(0)
			splitTarget := strings.Split(target, " ")
			if ep.RecordType == recordTypeSRV && len(splitTarget) == 2 {
				content = splitTarget[1]
				priority64, err := strconv.ParseInt(splitTarget[0], 10, 32)
				if err != nil {
					logger.Warnf("failed to parse priority from target '%s'", target)
				} else {
					priority = int32(priority64)
				}
			}
			record := sdk.NewRecord(recordName, ep.RecordType, content)
			ttl := int32(ep.RecordTTL)
			if ttl != 0 {
				record.SetTtl(ttl)
			}
			if priority != 0 {
				record.SetPriority(priority)
			}
			result = append(result, sdk.NewRecordCreate(*record))
		}
		return result
	})
	if err := recordsToCreate.ForEach(func(ep *endpoint.Endpoint, recordCreate *sdk.RecordCreate) error {
		zone := zt.FindZoneByDomainName(ep.DNSName)
		if !zone.HasId() {
			return fmt.Errorf("no zone found for domain '%s'", ep.DNSName)
		}
		err := p.client.CreateRecord(ctx, *zone.GetId(), *recordCreate)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (p *Provider) createZoneTree(ctx context.Context) (*ionos.ZoneTree[sdk.ZoneRead], error) {
	zt := ionos.NewZoneTree[sdk.ZoneRead]()
	var allZones []sdk.ZoneRead
	offset := int32(0)
	for {
		zoneReadList, err := p.client.GetZones(ctx, offset)
		if err != nil {
			return nil, err
		}
		if zoneReadList.HasItems() {
			items := *zoneReadList.GetItems()
			allZones = append(allZones, items...)
			offset += zoneReadLimit
			if len(items) < zoneReadLimit || offset >= zoneReadMaxCount {
				break
			}
		} else {
			break
		}
	}
	for _, zoneRead := range allZones {
		zoneName := *zoneRead.GetProperties().GetZoneName()
		if !p.BaseProvider.GetDomainFilter().IsConfigured() || p.BaseProvider.GetDomainFilter().Match(zoneName) {
			zt.AddZone(zoneRead, zoneName)
		}
	}
	return zt, nil
}

func extractRecordName(fqdn string, zone sdk.ZoneRead) string {
	zoneName := *zone.GetProperties().GetZoneName()
	partOfZoneName := strings.Index(fqdn, zoneName)
	if partOfZoneName == 0 {
		return ""
	}
	return fqdn[:partOfZoneName-1]
}
