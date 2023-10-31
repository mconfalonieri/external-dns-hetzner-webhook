package hetzner

/*
This file was MODIFIED from the original to use panta/go-hetzner-dns library
and to be used as a standalone webhook server.

Copyright 2023 Marco Confalonieri.
---
Original file:

Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	hdns "github.com/panta/go-hetzner-dns"
	log "github.com/sirupsen/logrus"
)

// apiClient is an abstraction of the REST API client.
type apiClient interface {
	GetZones(ctx context.Context, name string, searchName string, page int, perPage int) (*hdns.ZonesResponse, error)
	GetRecords(ctx context.Context, zone_id string, page int, perPage int) (*hdns.RecordsResponse, error)
	CreateRecord(ctx context.Context, record hdns.RecordRequest) (*hdns.RecordResponse, error)
	UpdateRecord(ctx context.Context, record hdns.RecordRequest) (*hdns.RecordResponse, error)
	DeleteRecord(ctx context.Context, recordId string) error
}

// HetznerProvider contains the logic for connecting to the Hetzner DNS API.
type HetznerProvider struct {
	provider.BaseProvider
	client           apiClient
	batchSize        int
	debug            bool
	dryRun           bool
	defaultTTL       int
	zoneIDNameMapper provider.ZoneIDName
	domainFilter     endpoint.DomainFilter
}

// NewHetznerProvider creates a new HetznerProvider instance.
func NewHetznerProvider(config *Configuration) (*HetznerProvider, error) {
	var logLevel log.Level
	if config.Debug {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	return &HetznerProvider{
		client: &hdns.Client{
			ApiKey: config.APIKey,
			Debug:  config.Debug,
		},
		batchSize:    config.BatchSize,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		domainFilter: GetDomainFilter(*config),
	}, nil
}

// hetznerChangeCreate contains a create request.
type hetznerChangeCreate struct {
	Domain  string
	Request *hdns.RecordRequest
}

// hetznerChangeUpdate contains an update request.
type hetznerChangeUpdate struct {
	Domain       string
	DomainRecord hdns.Record
	Request      *hdns.RecordRequest
}

// hetznerChangedelete contains a delete request.
type hetznerChangeDelete struct {
	Domain   string
	RecordID string
}

// hetznerChanges contains all changes to apply to DNS
type hetznerChanges struct {
	Creates []*hetznerChangeCreate
	Updates []*hetznerChangeUpdate
	Deletes []*hetznerChangeDelete
}

// Empty is true if there are no changes.
func (c *hetznerChanges) Empty() bool {
	return len(c.Creates) == 0 && len(c.Updates) == 0 && len(c.Deletes) == 0
}

// Zones returns the list of hosted zones.
func (p *HetznerProvider) Zones(ctx context.Context) ([]hdns.Zone, error) {
	result := []hdns.Zone{}

	log.Debug("Fetching all zones.")
	zones, err := p.fetchZones(ctx)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if p.domainFilter.Match(zone.Name) {
			log.Debugf("Adding fetched zone [%s]", zone.Name)
			result = append(result, zone)
		} else {
			log.Debugf("Discarding fetched zone [%s]", zone.Name)
		}
	}

	p.ensureZoneIDMappingPresent(zones)

	return result, nil
}

// AdjustEndpoints adjusts the endpoint conforming to Hetzner's requirements.
func (p HetznerProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjustedEndpoints := []*endpoint.Endpoint{}

	for _, ep := range endpoints {
		_, zoneName := p.zoneIDNameMapper.FindZone(ep.DNSName)
		adjustedTargets := endpoint.Targets{}
		for _, t := range ep.Targets {
			adjustedTarget, producedValidTarget := makeEndpointTarget(zoneName, t, ep.RecordType)
			if producedValidTarget {
				adjustedTargets = append(adjustedTargets, adjustedTarget)
			}
		}

		ep.Targets = adjustedTargets
		adjustedEndpoints = append(adjustedEndpoints, ep)
	}

	return adjustedEndpoints, nil
}

// mergeEndpointsByNameType merges endpoints with the same Name and Type into a
// single endpoint with multiple Targets.
func mergeEndpointsByNameType(endpoints []*endpoint.Endpoint) []*endpoint.Endpoint {
	endpointsByNameType := map[string][]*endpoint.Endpoint{}

	for _, e := range endpoints {
		key := fmt.Sprintf("%s-%s", e.DNSName, e.RecordType)
		endpointsByNameType[key] = append(endpointsByNameType[key], e)
	}

	// If no merge occurred, just return the existing endpoints.
	if len(endpointsByNameType) == len(endpoints) {
		return endpoints
	}

	// Otherwise, construct a new list of endpoints with the endpoints merged.
	var result []*endpoint.Endpoint
	for _, endpoints := range endpointsByNameType {
		dnsName := endpoints[0].DNSName
		recordType := endpoints[0].RecordType

		targets := make([]string, len(endpoints))
		for i, e := range endpoints {
			targets[i] = e.Targets[0]
		}

		e := endpoint.NewEndpoint(dnsName, recordType, targets...)
		e.RecordTTL = endpoints[0].RecordTTL
		result = append(result, e)
	}

	return result
}

// Records returns the list of records in a given zone.
func (p *HetznerProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		log.Debugf("Fetching all records from zone [%s].", zone.Name)
		records, err := p.fetchRecords(ctx, zone.ID)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			if provider.SupportedRecordType(string(r.Type)) {
				name := fmt.Sprintf("%s.%s", r.Name, zone.Name)

				// root name is identified by @ and should be
				// translated to zone name for the endpoint entry.
				if r.Name == "@" {
					name = zone.Name
				}
				log.Debugf("Adding endpoint [%s] of supported type %s.", name, r.Type)
				ep := endpoint.NewEndpoint(name, string(r.Type), r.Value)
				ep.RecordTTL = endpoint.TTL(r.TTL)
				endpoints = append(endpoints, ep)
			} else {
				log.Debugf("Discarding record [%s.%s] on unsupported type %s.", r.Name,
					zone.Name, r.Type)
			}
		}
	}

	// Merge endpoints with the same name and type (e.g., multiple A records for a single
	// DNS name) into one endpoint with multiple targets.
	endpoints = mergeEndpointsByNameType(endpoints)

	// Log the endpoints that were found.
	log.WithFields(log.Fields{
		"endpoints": endpoints,
	}).Debug("Endpoints generated from Hetzner DNS")

	return endpoints, nil
}

// fetchRecords fetches all records for a given zone.
func (p *HetznerProvider) fetchRecords(ctx context.Context, zoneID string) ([]hdns.Record, error) {
	resp, err := p.client.GetRecords(ctx, zoneID, 0, 0)
	if err != nil {
		return nil, err
	}
	return resp.Records, nil
}

// fetchZones fetches all the zones.
func (p *HetznerProvider) fetchZones(ctx context.Context) ([]hdns.Zone, error) {
	allZones := []hdns.Zone{}
	page := 1
	for {
		log.Debugf("Getting %d results from page %d", p.batchSize, page)
		resp, err := p.client.GetZones(ctx, "", "", page, p.batchSize)
		if err != nil {
			return nil, err
		}
		zones := resp.Zones
		allZones = append(allZones, zones...)

		if resp.Meta.Pagination.LastPage <= resp.Meta.Pagination.Page {
			break
		}

		page = resp.Meta.Pagination.Page + 1
	}
	log.Debugf("Fetched %d zones:", len(allZones))
	for _, z := range allZones {
		log.Debugf("- [ID:%s] %s", z.ID, z.Name)
	}

	return allZones, nil
}

// ensureZoneIDMappingPresent ensures that all the zone IDs are mapped.
func (p *HetznerProvider) ensureZoneIDMappingPresent(zones []hdns.Zone) {
	zoneIDNameMapper := provider.ZoneIDName{}
	for _, z := range zones {
		zoneIDNameMapper.Add(z.ID, z.Name)
	}
	p.zoneIDNameMapper = zoneIDNameMapper
}

// getRecordsByZoneID returns the records for a given zone.
func (p *HetznerProvider) getRecordsByZoneID(ctx context.Context) (map[string][]hdns.Record, provider.ZoneIDName, error) {
	recordsByZoneID := map[string][]hdns.Record{}

	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Fetch records for each zone
	for _, zone := range zones {
		records, err := p.fetchRecords(ctx, zone.ID)
		if err != nil {
			return nil, nil, err
		}

		recordsByZoneID[zone.ID] = append(recordsByZoneID[zone.ID], records...)
	}

	return recordsByZoneID, p.zoneIDNameMapper, nil
}

// makeEndpointName makes a endpoint name that conforms to Hetzner DNS
// requirements:
// - Records at root of the zone have `@` as the name
func makeEndpointName(domain, entryName, epType string) string {
	// Trim the domain off the name if present.
	adjustedName := strings.TrimSuffix(entryName, "."+domain)

	// Record at the root should be defined as @ instead of the full domain name.
	if adjustedName == domain {
		adjustedName = "@"
	}

	return adjustedName
}

// makeEndpointTarget makes a endpoint name that conforms to Hetzner DNS
// requirements:
// - Records at root of the zone have `@` as the name
// - A-Records should respect ignored networks and should only contain IPv4 entries
func makeEndpointTarget(domain, entryTarget, recordType string) (string, bool) {
	if domain == "" {
		return entryTarget, true
	}

	// Trim the trailing dot
	adjustedTarget := strings.TrimSuffix(entryTarget, ".")
	adjustedTarget = strings.TrimSuffix(adjustedTarget, "."+domain)

	return adjustedTarget, true
}

// submitChanges applies an instance of `hetznerChanges` to the Hetzner API.
func (p *HetznerProvider) submitChanges(ctx context.Context, changes *hetznerChanges) error {
	// return early if there is nothing to change
	if changes.Empty() {
		return nil
	}

	for _, d := range changes.Deletes {
		log.WithFields(log.Fields{
			"domain":   d.Domain,
			"recordID": d.RecordID,
		}).Debug("Deleting domain record")

		if p.dryRun {
			continue
		}

		err := p.client.DeleteRecord(ctx, d.RecordID)
		if err != nil {
			return err
		}
	}

	for _, c := range changes.Creates {
		ttl := c.Request.TTL
		log.WithFields(log.Fields{
			"domain":     c.Domain,
			"zoneID":     c.Request.ZoneID,
			"dnsName":    c.Request.Name,
			"recordType": c.Request.Type,
			"value":      c.Request.Value,
			"ttl":        ttl,
		}).Debug("Creating domain record")

		if p.dryRun {
			continue
		}

		_, err := p.client.CreateRecord(ctx, *c.Request)
		if err != nil {
			return err
		}
	}

	for _, u := range changes.Updates {
		ttl := u.Request.TTL
		log.WithFields(log.Fields{
			"domain":     u.Domain,
			"zoneID":     u.Request.ZoneID,
			"dnsName":    u.Request.Name,
			"recordType": u.Request.Type,
			"value":      u.Request.Value,
			"ttl":        ttl,
		}).Debug("Updating domain record")

		if p.dryRun {
			continue
		}

		_, err := p.client.UpdateRecord(ctx, *u.Request)
		if err != nil {
			return err
		}
	}

	return nil
}

func endpointsByZoneID(zoneIDNameMapper provider.ZoneIDName, endpoints []*endpoint.Endpoint) map[string][]*endpoint.Endpoint {
	endpointsByZoneID := make(map[string][]*endpoint.Endpoint)

	for _, ep := range endpoints {
		zoneID, _ := zoneIDNameMapper.FindZone(ep.DNSName)
		if zoneID == "" {
			log.Debugf("Skipping record %s because no hosted zone matching record DNS Name was detected", ep.DNSName)
			continue
		}
		endpointsByZoneID[zoneID] = append(endpointsByZoneID[zoneID], ep)
	}

	return endpointsByZoneID
}

func getMatchingDomainRecords(records []hdns.Record, zoneName string, ep *endpoint.Endpoint) []hdns.Record {
	var name string
	if ep.DNSName != zoneName {
		name = strings.TrimSuffix(ep.DNSName, "."+zoneName)
	} else {
		name = "@"
	}

	var result []hdns.Record
	for _, r := range records {
		if r.Name == name && string(r.Type) == ep.RecordType {
			result = append(result, r)
		}
	}
	return result
}

func getTTLFromEndpoint(ep *endpoint.Endpoint) (int, bool) {
	if ep.RecordTTL.IsConfigured() {
		return int(ep.RecordTTL), true
	}
	return -1, false
}

func processCreateActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	createsByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
	defaultTTL int,
) error {
	// Process endpoints that need to be created.
	for zoneID, endpoints := range createsByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(endpoints) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping domain, no creates found.")
			continue
		}

		records := recordsByZoneID[zoneName]

		for _, ep := range endpoints {
			// Warn if there are existing records since we expect to create only new records.
			matchingRecords := getMatchingDomainRecords(records, zoneName, ep)
			if len(matchingRecords) > 0 {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
				}).Warn("Preexisting records exist which should not exist for creation actions.")
			}

			ttl := defaultTTL
			configuredTTL, ttlIsSet := getTTLFromEndpoint(ep)
			if ttlIsSet {
				ttl = configuredTTL
			}

			for _, target := range ep.Targets {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
					"target":     target,
					"ttl":        ttl,
				}).Warn("Creating new target")

				changes.Creates = append(changes.Creates, &hetznerChangeCreate{
					Domain: zoneName,
					Request: &hdns.RecordRequest{
						Name:   makeEndpointName(zoneName, ep.DNSName, ep.RecordType),
						TTL:    ttl,
						Type:   ep.RecordType,
						Value:  target,
						ZoneID: zoneID,
					},
				})
			}
		}
	}

	return nil
}

func processUpdateActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	updatesByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
	defaultTTL int,
) error {
	// Generate creates and updates based on existing
	for zoneID, updates := range updatesByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(updates) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no updates found.")
			continue
		}

		records := recordsByZoneID[zoneID]
		log.WithFields(log.Fields{
			"zoneName": zoneName,
			"records":  records,
		}).Debug("Records for domain")

		for _, ep := range updates {
			matchingRecords := getMatchingDomainRecords(records, zoneName, ep)

			log.WithFields(log.Fields{
				"endpoint":        ep,
				"matchingRecords": matchingRecords,
			}).Debug("matching records")

			if len(matchingRecords) == 0 {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
				}).Warn("Planning an update but no existing records found.")
			}

			matchingRecordsByTarget := map[string]hdns.Record{}
			for _, r := range matchingRecords {
				matchingRecordsByTarget[r.Value] = r
			}

			ttl := defaultTTL
			configuredTTL, ttlIsSet := getTTLFromEndpoint(ep)
			if ttlIsSet {
				ttl = configuredTTL
			}

			// Generate create and delete actions based on existence of a record for each target.
			for _, target := range ep.Targets {
				if record, ok := matchingRecordsByTarget[target]; ok {
					log.WithFields(log.Fields{
						"zoneName":   zoneName,
						"dnsName":    ep.DNSName,
						"recordType": ep.RecordType,
						"target":     target,
						"ttl":        ttl,
					}).Warn("Updating existing target")

					changes.Updates = append(changes.Updates, &hetznerChangeUpdate{
						Domain:       zoneName,
						DomainRecord: record,
						Request: &hdns.RecordRequest{
							Name:   makeEndpointName(zoneName, ep.DNSName, ep.RecordType),
							TTL:    ttl,
							Type:   ep.RecordType,
							Value:  target,
							ZoneID: zoneID,
						},
					})

					delete(matchingRecordsByTarget, target)
				} else {
					// Record did not previously exist, create new 'target'
					log.WithFields(log.Fields{
						"zoneName":   zoneName,
						"dnsName":    ep.DNSName,
						"recordType": ep.RecordType,
						"target":     target,
						"ttl":        ttl,
					}).Warn("No target to update - creating new target")

					changes.Creates = append(changes.Creates, &hetznerChangeCreate{
						Domain: zoneName,
						Request: &hdns.RecordRequest{
							Name:   makeEndpointName(zoneName, ep.DNSName, ep.RecordType),
							TTL:    ttl,
							Type:   ep.RecordType,
							Value:  target,
							ZoneID: zoneID,
						},
					})
				}
			}

			// Any remaining records have been removed, delete them
			for _, record := range matchingRecordsByTarget {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
					"target":     record.Value,
				}).Warn("Deleting target")

				changes.Deletes = append(changes.Deletes, &hetznerChangeDelete{
					Domain:   zoneName,
					RecordID: record.ID,
				})
			}
		}
	}

	return nil
}

func processDeleteActions(
	zoneIDNameMapper provider.ZoneIDName,
	recordsByZoneID map[string][]hdns.Record,
	deletesByZoneID map[string][]*endpoint.Endpoint,
	changes *hetznerChanges,
) error {
	// Generate delete actions for each deleted endpoint.
	for zoneID, deletes := range deletesByZoneID {
		zoneName := zoneIDNameMapper[zoneID]
		if len(deletes) == 0 {
			log.WithFields(log.Fields{
				"zoneName": zoneName,
			}).Debug("Skipping Zone, no deletes found.")
			continue
		}

		records := recordsByZoneID[zoneID]

		for _, ep := range deletes {
			matchingRecords := getMatchingDomainRecords(records, zoneName, ep)

			if len(matchingRecords) == 0 {
				log.WithFields(log.Fields{
					"zoneName":   zoneName,
					"dnsName":    ep.DNSName,
					"recordType": ep.RecordType,
				}).Warn("Records to delete not found.")
			}

			for _, record := range matchingRecords {
				doDelete := false
				for _, t := range ep.Targets {
					v1 := t
					v2 := record.Value
					if ep.RecordType == endpoint.RecordTypeCNAME {
						v1 = strings.TrimSuffix(t, ".")
						v2 = strings.TrimSuffix(t, ".")
					}
					if v1 == v2 {
						doDelete = true
					}
				}

				if doDelete {
					changes.Deletes = append(changes.Deletes, &hetznerChangeDelete{
						Domain:   zoneName,
						RecordID: record.ID,
					})
				}
			}
		}
	}

	return nil
}

// ApplyChanges applies the given set of generic changes to the provider.
func (p *HetznerProvider) ApplyChanges(ctx context.Context, planChanges *plan.Changes) error {
	log.Debug("Applying changes.")
	// TODO: This should only retrieve zones affected by the given `planChanges`.
	recordsByZoneID, zoneIDNameMapper, err := p.getRecordsByZoneID(ctx)
	if err != nil {
		return err
	}

	createsByZoneID := endpointsByZoneID(zoneIDNameMapper, planChanges.Create)
	updatesByZoneID := endpointsByZoneID(zoneIDNameMapper, planChanges.UpdateNew)
	deletesByZoneID := endpointsByZoneID(zoneIDNameMapper, planChanges.Delete)

	var changes hetznerChanges

	if err := processCreateActions(zoneIDNameMapper, recordsByZoneID, createsByZoneID, &changes, p.defaultTTL); err != nil {
		return err
	}

	if err := processUpdateActions(zoneIDNameMapper, recordsByZoneID, updatesByZoneID, &changes, p.defaultTTL); err != nil {
		return err
	}

	if err := processDeleteActions(zoneIDNameMapper, recordsByZoneID, deletesByZoneID, &changes); err != nil {
		return err
	}

	return p.submitChanges(ctx, &changes)
}
