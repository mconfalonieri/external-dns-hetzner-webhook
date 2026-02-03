/*
 * Provider - class and functions that handle the connection to Hetzner DNS.
 *
 * This file was MODIFIED from the original provider to be used as a standalone
 * webhook server.
 *
 * Copyright 2026 Marco Confalonieri.
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
package hetznercloud

import (
	"context"
	"fmt"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner"
	"external-dns-hetzner-webhook/internal/metrics"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	log "github.com/sirupsen/logrus"
)

// logFatalf is a mockable call to log.Fatalf
var logFatalf = log.Fatalf

// changesRunner is the general interface for applying changes.
type changesRunner interface {
	// AddChangeCreate adds a new creation entry to the current object.
	AddChangeCreate(zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts)
	// AddChangeUpdate adds a new update entry to the current object.
	AddChangeUpdate(rrset *hcloud.ZoneRRSet, ttlOpts *hcloud.ZoneRRSetChangeTTLOpts, recordsOpts *hcloud.ZoneRRSetSetRecordsOpts, updateOpts *hcloud.ZoneRRSetUpdateOpts)
	// AddChangeDelete adds a new delete entry to the current object.
	AddChangeDelete(rrset *hcloud.ZoneRRSet)
	// ApplyChanges applies the planned changes using dnsClient.
	ApplyChanges(ctx context.Context) error
	// GetSlash returns the current slash escape sequence and a boolean that
	// determines if labels are supported by the implementation.
	GetSlash() (string, bool)
}

// HetznerProvider implements ExternalDNS' provider.Provider interface for
// Hetzner.
type HetznerProvider struct {
	provider.BaseProvider
	client            apiClient
	batchSize         int
	debug             bool
	dryRun            bool
	defaultTTL        int
	zoneIDNameMapper  zoneIDName
	domainFilter      *endpoint.DomainFilter
	slashEscSeq       string
	maxFailCount      int
	failCount         int
	zoneCacheDuration time.Duration
	zoneCacheUpdate   time.Time
	zoneCache         []*hcloud.Zone
	bulkMode         bool
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

	client, err := NewHetznerCloud(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate cloud DNS provider: %w", err)
	}

	var msg string
	if config.MaxFailCount > 0 {
		msg = fmt.Sprintf("Configuring cloud DNS provider with maximum fail count of %d", config.MaxFailCount)
	} else {
		msg = "Configuring cloud DNS provider without maximum fail count"
	}
	log.Info(msg)

	if config.BulkMode {
		log.Info("Experimental BULK_MODE activated: changes will use import/export endpoints.")
	}

	zcTTL := time.Duration(int64(config.ZoneCacheTTL) * int64(time.Second))
	zcUpdate := time.Now()

	if zcTTL > 0 {
		log.Infof("Zone cache enabled. TTL=%ds.", config.ZoneCacheTTL)
	} else {
		log.Info("Zone cache disabled in configuration.")
	}

	return &HetznerProvider{
		client:            client,
		batchSize:         config.BatchSize,
		debug:             config.Debug,
		dryRun:            config.DryRun,
		defaultTTL:        config.DefaultTTL,
		domainFilter:      hetzner.GetDomainFilter(*config),
		slashEscSeq:       config.SlashEscSeq,
		maxFailCount:      config.MaxFailCount,
		zoneCacheDuration: zcTTL,
		zoneCacheUpdate:   zcUpdate,
    bulkMode:     config.BulkMode,
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
func (p *HetznerProvider) Zones(ctx context.Context) ([]*hcloud.Zone, error) {
	now := time.Now()
	if now.Before(p.zoneCacheUpdate) && p.zoneCache != nil {
		nextUpdate := int(p.zoneCacheUpdate.Sub(now).Seconds())
		log.Debugf("Using cached zones. The cache expires in %d seconds.", nextUpdate)
		return p.zoneCache, nil
	}
	metrics := metrics.GetOpenMetricsInstance()
	result := []*hcloud.Zone{}

	zones, err := fetchZones(ctx, p.client, p.batchSize)
	if err != nil {
		log.Errorf("Got an error while fetching zones: %s", err.Error())
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

	log.Debugf("Got %d zones, filtered out %d zones.", len(zones), filteredOutZones)
	p.ensureZoneIDMappingPresent(zones)
	p.zoneCache = result
	p.zoneCacheUpdate = now.Add(p.zoneCacheDuration)

	return result, nil
}

// AdjustEndpoints adjusts the endpoints according to the provider
// requirements.
func (p HetznerProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjustedEndpoints := []*endpoint.Endpoint{}

	for _, ep := range endpoints {
		_, zone := p.zoneIDNameMapper.FindZone(ep.DNSName)
		var adjustedTargets endpoint.Targets
		if zone == nil {
			adjustedTargets = ep.Targets
		} else {
			var err error
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
		p.incFailCount()
		return nil, err
	}
	p.resetFailCount()

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
			// Use our own IsSupportedRecordType instead of provider.SupportedRecordType
			// because the SDK function doesn't include MX in its hardcoded list.
			if hetzner.IsSupportedRecordType(string(rrset.Type)) {
				ep := createEndpointFromRecord(p.slashEscSeq, rrset)
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
	zoneIDNameMapper := zoneIDName{}
	for _, z := range zones {
		zoneIDNameMapper.Add(z)
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

// getChangesRunner returns the appropriate changesRunner depending on the
// BULK_MODE flag.
func (p HetznerProvider) getChangesRunner() changesRunner {
	if p.bulkMode {
		return NewBulkChanges(p.client, p.dryRun, p.defaultTTL, p.slashEscSeq)
	} else {
		return NewHetznerChanges(p.client, p.dryRun, p.defaultTTL, p.slashEscSeq)
	}
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

	changes := p.getChangesRunner()

	processCreateActions(p.zoneIDNameMapper, rrSetsByZoneID, createsByZoneID, changes)
	processUpdateActions(p.zoneIDNameMapper, rrSetsByZoneID, updatesByZoneID, changes)
	processDeleteActions(p.zoneIDNameMapper, rrSetsByZoneID, deletesByZoneID, changes)

	return changes.ApplyChanges(ctx)
}

// GetDomainFilter returns the domain filter
func (p HetznerProvider) GetDomainFilter() endpoint.DomainFilterInterface {
	return p.domainFilter
}
