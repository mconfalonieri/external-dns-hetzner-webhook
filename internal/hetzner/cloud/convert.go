/*
 * HCloud - Conversion utilities.
 *
 * Copyright 2023 Marco Confalonieri.
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
package cloud

import (
	"external-dns-hetzner-webhook/internal/hetzner/model"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// getHZoneFromID returns a hcloud zone from an ID in string format.
func getHZoneFromID(zoneID string) (*hcloud.Zone, error) {
	id, err := strconv.ParseInt(zoneID, 16, 64)
	if err != nil {
		return nil, err
	}
	return &hcloud.Zone{ID: id}, nil
}

// getZone converts an hcloud zone to a model one.
func getZone(zone hcloud.Zone) model.Zone {
	return model.Zone{
		ID:      strconv.FormatInt(zone.ID, 16),
		Created: zone.Created,
		Name:    zone.Name,
		TTL:     zone.TTL,
	}
}

// getPZoneArray converts an array of hcloud zone pointers to an array of model
// zones.
func getPZoneArray(zones []*hcloud.Zone) []model.Zone {
	mZones := make([]model.Zone, len(zones))
	for i, z := range zones {
		mZones[i] = getZone(*z)
	}
	return mZones
}

func getRecord(record hcloud.ZoneRRSet) model.Record {
	var z *model.Zone = nil
	if record.Zone != nil {
		zone := getZone(*record.Zone)
		z = &zone
	}
	return model.Record{
		ID: record.ID,
	}
}
