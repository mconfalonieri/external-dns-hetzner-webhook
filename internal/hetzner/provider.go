/*
 * This file was MODIFIED from the original provider to be used as a standalone
 * webhook server.
 *
 * Copyright 2023 Marco Confalonieri.
 * Copyright 2017 The Kubernetes Authors.
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
package hetzner

import (
	"context"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	"github.com/bsm/openmetrics"
	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// HetznerProvider implements ExternalDNS' provider.Provider interface for
// Hetzner.
type HetznerProvider struct {
	provider.BaseProvider
	client           apiClient
	batchSize        int
	debug            bool
	dryRun           bool
	defaultTTL       int
	zoneIDNameMapper provider.ZoneIDName
	domainFilter     endpoint.DomainFilter
	reg              *openmetrics.Registry
}

// NewHetznerProvider creates a new HetznerProvider instance.
func NewHetznerProvider(config *Configuration, reg *openmetrics.Registry) (*HetznerProvider, error) {
	var logLevel log.Level
	if config.Debug {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	return &HetznerProvider{
		client:       NewHetznerDNS(config.APIKey),
		batchSize:    config.BatchSize,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		domainFilter: GetDomainFilter(*config),
		reg:          reg,
	}, nil
}

// Zones returns the list of the hosted DNS zones.
// If a domain filter is set, it only returns the zones that match it.
func (p *HetznerProvider) Zones(ctx context.Context) ([]hdns.Zone, error) {
	result := []hdns.Zone{}

	zones, err := fetchZones(ctx, p.client, p.batchSize)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if p.domainFilter.Match(zone.Name) {
			result = append(result, zone)
		}
	}

	p.ensureZoneIDMappingPresent(zones)

	return result, nil
}

// AdjustEndpoints adjusts the endpoints according to the provider
// requirements.
func (p HetznerProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjustedEndpoints := []*endpoint.Endpoint{}

	for _, ep := range endpoints {
		_, zoneName := p.zoneIDNameMapper.FindZone(ep.DNSName)
		adjustedTargets := endpoint.Targets{}
		for _, t := range ep.Targets {
			adjustedTarget := makeEndpointTarget(zoneName, t, ep.RecordType)
			adjustedTargets = append(adjustedTargets, adjustedTarget)
		}

		ep.Targets = adjustedTargets
		adjustedEndpoints = append(adjustedEndpoints, ep)
	}

	return adjustedEndpoints, nil
}

// Records returns the list of records in all zones as a slice of endpoints.
func (p *HetznerProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		records, err := fetchRecords(ctx, zone.ID, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}

		// Add only endpoints from supported types.
		for _, r := range records {
			if provider.SupportedRecordType(string(r.Type)) {
				ep := createEndpointFromRecord(r)
				endpoints = append(endpoints, ep)
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

// ensureZoneIDMappingPresent prepares the zoneIDNameMapper, that associates
// each ZoneID woth the zone name.
func (p *HetznerProvider) ensureZoneIDMappingPresent(zones []hdns.Zone) {
	zoneIDNameMapper := provider.ZoneIDName{}
	for _, z := range zones {
		zoneIDNameMapper.Add(z.ID, z.Name)
	}
	p.zoneIDNameMapper = zoneIDNameMapper
}

// getRecordsByZoneID returns a map that associates each ZoneID with the
// records contained in that zone.
func (p *HetznerProvider) getRecordsByZoneID(ctx context.Context) (map[string][]hdns.Record, error) {
	recordsByZoneID := map[string][]hdns.Record{}

	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch records for each zone
	for _, zone := range zones {
		records, err := fetchRecords(ctx, zone.ID, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}

		recordsByZoneID[zone.ID] = append(recordsByZoneID[zone.ID], records...)
	}

	return recordsByZoneID, nil
}

// ApplyChanges applies the given set of generic changes to the provider.
func (p *HetznerProvider) ApplyChanges(ctx context.Context, planChanges *plan.Changes) error {
	if !planChanges.HasChanges() {
		return nil
	}

	recordsByZoneID, err := p.getRecordsByZoneID(ctx)
	if err != nil {
		return err
	}

	createsByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Create)
	updatesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.UpdateNew)
	deletesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Delete)

	changes := hetznerChanges{
		dryRun: p.dryRun,
	}

	processCreateActions(p.zoneIDNameMapper, recordsByZoneID, createsByZoneID, &changes)
	processUpdateActions(p.zoneIDNameMapper, recordsByZoneID, updatesByZoneID, &changes)
	processDeleteActions(p.zoneIDNameMapper, recordsByZoneID, deletesByZoneID, &changes)

	return changes.ApplyChanges(ctx, p.client)
}
