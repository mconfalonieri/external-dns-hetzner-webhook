/*
 * Provider - class and functions that handle the connection to Hetzner DNS.
 *
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
	"strconv"

	"external-dns-hetzner-webhook/internal/metrics"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
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
	domainFilter     *endpoint.DomainFilter
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
		client:       NewHetznerCloud(config.APIKey),
		batchSize:    config.BatchSize,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		domainFilter: GetDomainFilter(*config),
	}, nil
}

// Zones returns the list of the hosted DNS zones.
// If a domain filter is set, it only returns the zones that match it.
func (p *HetznerProvider) Zones(ctx context.Context) ([]*hcloud.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	result := []*hcloud.Zone{}

	zones, err := fetchZones(ctx, p.client, p.batchSize)
	if err != nil {
		return nil, err
	}

	filteredOutZones := 0
	for _, zone := range zones {
		if p.domainFilter.Match(zone.Name) {
			result = append(result, zone)
		} else {
			filteredOutZones++
		}
	}
	metrics.SetFilteredOutZones(filteredOutZones)

	p.ensureZoneIDMappingPresent(zones)

	return result, nil
}

// AdjustEndpoints adjusts the endpoints according to the provider
// requirements.
func (p HetznerProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjustedEndpoints := []*endpoint.Endpoint{}

	for _, ep := range endpoints {
		_, zoneName := p.zoneIDNameMapper.FindZone(ep.DNSName)
		var adjustedTargets endpoint.Targets
		if zoneName == "" {
			adjustedTargets = ep.Targets
		} else {
			var err error = nil
			if adjustedTargets, err = adjustEndpointTargets(ep.Targets); err != nil {
				return nil, err
			}
		}
		ep.Targets = adjustedTargets
		adjustedEndpoints = append(adjustedEndpoints, ep)
	}
	return adjustedEndpoints, nil
}

// logDebugEndpoints logs every endpoint as a a line.
func logDebugEndpoints(endpoints []*endpoint.Endpoint) {
	for idx, ep := range endpoints {
		log.WithFields(getEndpointLogFields(ep)).Debugf("Endpoint %d", idx)
	}
}

// Records returns the list of records in all zones as a slice of endpoints.
func (p *HetznerProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		rrsets, err := fetchRecords(ctx, zone, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}

		skippedRecords := 0
		// Add only endpoints from supported types.
		for _, rrset := range rrsets {
			// Ensure the record has all the required zone information
			rrset.Zone = zone
			if provider.SupportedRecordType(string(rrset.Type)) {
				ep := createEndpointFromRecord(rrset)
				endpoints = append(endpoints, ep)
			} else {
				skippedRecords++
			}
		}
		m := metrics.GetOpenMetricsInstance()
		m.SetSkippedRecords(zone.Name, skippedRecords)
	}

	// Log the endpoints that were found.
	if p.debug {
		log.Debugf("Returning %d endpoints.", len(endpoints))
		logDebugEndpoints(endpoints)
	}

	return endpoints, nil
}

// ensureZoneIDMappingPresent prepares the zoneIDNameMapper, that associates
// each ZoneID woth the zone name.
func (p *HetznerProvider) ensureZoneIDMappingPresent(zones []*hcloud.Zone) {
	zoneIDNameMapper := provider.ZoneIDName{}
	for _, z := range zones {
		zoneID := strconv.FormatInt(z.ID, 10)
		zoneIDNameMapper.Add(zoneID, z.Name)
	}
	p.zoneIDNameMapper = zoneIDNameMapper
}

// getRRSetsByZoneID returns a map that associates each ZoneID with the
// RRSets contained in that zone.
func (p *HetznerProvider) getRRSetsByZoneID(ctx context.Context) (map[int64][]*hcloud.ZoneRRSet, error) {
	rrSetsByZoneID := make(map[int64][]*hcloud.ZoneRRSet, 0)

	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch records for each zone
	for _, zone := range zones {
		rrsets, err := fetchRecords(ctx, zone, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}
		rrSetsByZoneID[zone.ID] = rrsets
	}

	return rrSetsByZoneID, nil
}

// ApplyChanges applies the given set of generic changes to the provider.
func (p *HetznerProvider) ApplyChanges(ctx context.Context, planChanges *plan.Changes) error {
	if !planChanges.HasChanges() {
		return nil
	}

	rrSetsByZoneID, err := p.getRRSetsByZoneID(ctx)
	if err != nil {
		return err
	}

	log.Debug("Preparing creates")
	createsByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Create)
	log.Debug("Preparing updates")
	updatesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.UpdateNew)
	log.Debug("Preparing deletes")
	deletesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Delete)

	changes := hetznerChanges{
		dryRun:     p.dryRun,
		defaultTTL: p.defaultTTL,
	}

	processCreateActions(p.zoneIDNameMapper, rrSetsByZoneID, createsByZoneID, &changes)
	processUpdateActions(p.zoneIDNameMapper, rrSetsByZoneID, updatesByZoneID, &changes)
	processDeleteActions(p.zoneIDNameMapper, rrSetsByZoneID, deletesByZoneID, &changes)

	return changes.ApplyChanges(ctx, p.client)
}
