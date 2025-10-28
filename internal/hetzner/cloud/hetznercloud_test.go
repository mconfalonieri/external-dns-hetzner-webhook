/*
 * HetznerDNS - Common test routines for the hetzner package.
 *
 * Copyright 2024 Marco Confalonieri.
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
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

// testTTL is a test ttl.
var testTTL = 7200

// zonesResponse simulates a response that returns a list of zones.
type zonesResponse struct {
	zones []*hcloud.Zone
	resp  *hcloud.Response
	err   error
}

// rrSetsResponse simulates a response that returns a list of records.
type rrSetsResponse struct {
	rrsets []*hcloud.ZoneRRSet
	resp   *hcloud.Response
	err    error
}

// actionResponse simulates a response that returns the results of an action.
type actionResponse struct {
	action *hcloud.Action
	resp   *hcloud.Response
	err    error
}

// deleteRRSetResponse simulates a response to a record deletion request.
type deleteRRSetResponse struct {
	result hcloud.ZoneRRSetDeleteResult
	resp   *hcloud.Response
	err    error
}

// mockClientState keeps track of which methods were called.
type mockClientState struct {
	GetZonesCalled           bool
	GetRRSetsCalled          bool
	CreateRRSetCalled        bool
	UpdateRRSetTTLCalled     bool
	UpdateRRSetRecordsCalled bool
	DeleteRRSetCalled        bool
}

// mockClient represents the mock client used to simulate calls to the DNS API.
type mockClient struct {
	getZones           zonesResponse
	getRRSets          rrSetsResponse
	createRRSet        actionResponse
	updateRRSetTTL     actionResponse
	updateRRSetRecords actionResponse
	deleteRRSet        deleteRRSetResponse
	filterRRSetsByZone bool
	state              mockClientState
}

// GetState returns the internal state
func (m mockClient) GetState() mockClientState {
	return m.state
}

// GetZones simulates a request to get a list of zones.
func (m *mockClient) GetZones(ctx context.Context, opts hcloud.ZoneListOpts) ([]*hcloud.Zone, *hcloud.Response, error) {
	r := m.getZones
	m.state.GetZonesCalled = true
	return r.zones, r.resp, r.err
}

// filterRecordsByZone filters the records, returning only those for the selected zone.
func filterRecordsByZone(r rrSetsResponse, zoneID int64) []*hcloud.ZoneRRSet {
	rrsets := make([]*hcloud.ZoneRRSet, 0)
	for _, rrset := range r.rrsets {
		if rrset != nil && rrset.Zone.ID == zoneID {
			rrsets = append(rrsets, rrset)
		}
	}
	return rrsets
}

// GetRRSets simulates a request to get a list of RRSets for a given zone.
func (m *mockClient) GetRRSets(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetListOpts) ([]*hcloud.ZoneRRSet, *hcloud.Response, error) {
	r := m.getRRSets
	m.state.GetRRSetsCalled = true
	var rrsets []*hcloud.ZoneRRSet
	if m.filterRRSetsByZone {
		rrsets = filterRecordsByZone(r, zone.ID)
		r.resp.Meta.Pagination.TotalEntries = len(rrsets) // "smart" handling
	} else {
		rrsets = r.rrsets
	}
	return rrsets, r.resp, r.err
}

// CreateRRSet simulates a request to create a DNS record.
func (m *mockClient) CreateRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetAddRecordsOpts) (*hcloud.Action, *hcloud.Response, error) {
	r := m.createRRSet
	m.state.CreateRRSetCalled = true
	return r.action, r.resp, r.err
}

// UpdateRRSetTTL simulates a request to update a DNS record.
func (m *mockClient) UpdateRRSetTTL(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetChangeTTLOpts) (*hcloud.Action, *hcloud.Response, error) {
	r := m.updateRRSetTTL
	m.state.UpdateRRSetTTLCalled = true
	return r.action, r.resp, r.err
}

// UpdateRRSetRecords simulates a request to update a DNS record.
func (m *mockClient) UpdateRRSetRecords(ctx context.Context, rrset *hcloud.ZoneRRSet, opts hcloud.ZoneRRSetSetRecordsOpts) (*hcloud.Action, *hcloud.Response, error) {
	r := m.updateRRSetRecords
	m.state.UpdateRRSetRecordsCalled = true
	return r.action, r.resp, r.err
}

// DeleteRRSet simulates a request to delete a DNS record.
func (m *mockClient) DeleteRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet) (hcloud.ZoneRRSetDeleteResult, *hcloud.Response, error) {
	r := m.deleteRRSet
	m.state.DeleteRRSetCalled = true
	return r.result, r.resp, r.err
}

// assertError checks if an error is thrown when expected.
func assertError(t *testing.T, expected, actual error) bool {
	var expError bool
	if expected == nil {
		assert.Nil(t, actual)
		expError = false
	} else {
		assert.EqualError(t, actual, expected.Error())
		expError = true
	}
	return expError
}
