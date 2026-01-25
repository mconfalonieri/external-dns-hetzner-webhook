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
package hetznerdns

import (
	"context"
	"fmt"

	"external-dns-hetzner-webhook/internal/hetzner"
	"external-dns-hetzner-webhook/internal/metrics"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	log "github.com/sirupsen/logrus"
)

// logFatalf is a mockable call to log.Fatalf
var logFatalf = log.Fatalf

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
	maxFailCount     int
	failCount        int
}

// NewHetznerProvider creates a new HetznerProvider instance.
func NewHetznerProvider(config *hetzner.Configuration) (*HetznerProvider, error) {
	var logLevel log.Level
	if config.Debug {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	client, err := NewHetznerDNS(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate legacy DNS provider: %w", err)
	}

	var msg string
	if config.MaxFailCount > 0 {
		msg = fmt.Sprintf("Configuring legacy DNS provider with maximum fail count of %d", config.MaxFailCount)
	} else {
		msg = "Configuring legacy DNS provider without maximum fail count"
	}
	log.Info(msg)

	return &HetznerProvider{
		client:       client,
		batchSize:    config.BatchSize,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		domainFilter: hetzner.GetDomainFilter(*config),
		maxFailCount: config.MaxFailCount,
	}, nil
}

// incFailCount increments the fail count and exit if necessary.
func (p *HetznerProvider) incFailCount() {
	if p.maxFailCount <= 0 {
		return
	}
	p.failCount++
	if p.failCount >= p.maxFailCount {
		logFatalf("Failure count reached %d. Shutting down container.", p.failCount)
	}
}

// resetFailCount resets the fail count.
func (p *HetznerProvider) resetFailCount() {
	if p.maxFailCount <= 0 {
		return
	}
	p.failCount = 0
}

// Zones returns the list of the hosted DNS zones.
// If a domain filter is set, it only returns the zones that match it.
func (p *HetznerProvider) Zones(ctx context.Context) ([]hdns.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	result := []hdns.Zone{}

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
		p.incFailCount()
		return nil, err
	}
	p.resetFailCount()

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		records, err := fetchRecords(ctx, zone.ID, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}

		skippedRecords := 0
		// Add only endpoints from supported types.
		for _, r := range records {
			// Ensure the record has all the required zone information
			r.Zone = &zone
			// Use our own IsSupportedRecordType instead of provider.SupportedRecordType
			// because the SDK function doesn't include MX in its hardcoded list.
			if hetzner.IsSupportedRecordType(string(r.Type)) {
				ep := createEndpointFromRecord(r)
				endpoints = append(endpoints, ep)
			} else {
				skippedRecords++
			}
		}
		m := metrics.GetOpenMetricsInstance()
		m.SetSkippedRecords(zone.Name, skippedRecords)
	}

	// Merge endpoints with the same name and type (e.g., multiple A records for a single
	// DNS name) into one endpoint with multiple targets.
	endpoints = mergeEndpointsByNameType(endpoints)

	// Log the endpoints that were found.
	if p.debug {
		log.Debugf("Returning %d endpoints.", len(endpoints))
		logDebugEndpoints(endpoints)
	}

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
		// Add full zone information
		zonedRecords := []hdns.Record{}
		for _, r := range records {
			r.Zone = &zone
			zonedRecords = append(zonedRecords, r)
		}
		recordsByZoneID[zone.ID] = append(recordsByZoneID[zone.ID], zonedRecords...)
	}

	return recordsByZoneID, nil
}

// ApplyChanges applies the given set of generic changes to the provider.
func (p HetznerProvider) ApplyChanges(ctx context.Context, planChanges *plan.Changes) error {
	if !planChanges.HasChanges() {
		return nil
	}

	recordsByZoneID, err := p.getRecordsByZoneID(ctx)
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

	processCreateActions(p.zoneIDNameMapper, recordsByZoneID, createsByZoneID, &changes)
	processUpdateActions(p.zoneIDNameMapper, recordsByZoneID, updatesByZoneID, &changes)
	processDeleteActions(p.zoneIDNameMapper, recordsByZoneID, deletesByZoneID, &changes)

	return changes.ApplyChanges(ctx, p.client)
}

// GetDomainFilter returns the domain filter
func (p HetznerProvider) GetDomainFilter() endpoint.DomainFilterInterface {
	return p.domainFilter
}
