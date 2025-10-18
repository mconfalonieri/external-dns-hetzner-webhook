/*
 * Conversion utilities.
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
package dns

import (
	"time"

	"external-dns-hetzner-webhook/internal/hetzner/model"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
	"github.com/jobstoit/hetzner-dns-go/dns/schema"
)

// getDNSListOpts converts the listing options to the library format.
func getDNSListOpts(opts model.ListOpts) hdns.ListOpts {
	return hdns.ListOpts{
		Page:    opts.PageIdx,
		PerPage: opts.ItemsPerPage,
	}
}

// getDNSRecordListOpts converts the zone listing options to the library format.
func getDNSRecordListOpts(opts model.RecordListOpts) hdns.RecordListOpts {
	return hdns.RecordListOpts{
		ListOpts: getDNSListOpts(opts.ListOpts),
		ZoneID:   opts.ZoneID,
	}
}

// getDNSZoneListOpts converts the zone listing options to the library format.
func getDNSZoneListOpts(opts model.ZoneListOpts) hdns.ZoneListOpts {
	return hdns.ZoneListOpts{
		ListOpts:   getDNSListOpts(opts.ListOpts),
		Name:       opts.Name,
		SearchName: opts.SearchName,
	}
}

// getDNSZone converts a zone to the library format.
func getDNSZone(zone model.Zone) hdns.Zone {
	return hdns.Zone{
		ID:       zone.ID,
		Created:  schema.HdnsTime(zone.Created),
		Modified: schema.HdnsTime(zone.Modified),
		Name:     zone.Name,
		Ttl:      zone.TTL,
	}
}

// getZone converts a library zone to the model format.
func getZone(zone hdns.Zone) model.Zone {
	return model.Zone{
		ID:       zone.ID,
		Created:  time.Time(zone.Created),
		Modified: time.Time(zone.Modified),
		Name:     zone.Name,
		TTL:      zone.Ttl,
	}
}

// getPZoneArray converts an array of pointers to library zones to an array of
// model zones.
func getPZoneArray(zones []*hdns.Zone) []model.Zone {
	mZones := make([]model.Zone, len(zones))
	for i, z := range zones {
		mZones[i] = getZone(*z)
	}
	return mZones
}

// getDNSTtl converts the TTL value to a pointer.
func getDNSTtl(ttl int) *int {
	if ttl < 0 {
		return nil
	}
	libTTL := ttl
	return &libTTL
}

// getRecord converts a library record to a common model record.
func getRecord(record hdns.Record) model.Record {
	z := getZone(*record.Zone)
	return model.Record{
		ID:       record.ID,
		Name:     record.Name,
		Created:  time.Time(record.Created),
		Modified: time.Time(record.Modified),
		Zone:     &z,
		Type:     string(record.Type),
		Value:    record.Value,
		TTL:      record.Ttl,
	}
}

// getPRecordArray converts an array of pointers to library records to an array
// of model records.
func getPRecordArray(records []*hdns.Record) []model.Record {
	mRecords := make([]model.Record, len(records))
	for i, r := range records {
		mRecords[i] = getRecord(*r)
	}
	return mRecords
}

// getDNSRecordCreateOpts converts a record for creation through library.
func getDNSRecordCreateOpts(record model.Record) hdns.RecordCreateOpts {
	var z *hdns.Zone = nil
	if record.Zone != nil {
		dnsZone := getDNSZone(*record.Zone)
		z = &dnsZone
	}
	return hdns.RecordCreateOpts{
		Name:  record.Name,
		Zone:  z,
		Type:  hdns.RecordType(record.Type),
		Value: record.Value,
		Ttl:   getDNSTtl(record.TTL),
	}
}

// getDNSRecordUpdateOpts converts a record for update through library.
func getDNSRecordUpdateOpts(record model.Record) hdns.RecordUpdateOpts {
	var z *hdns.Zone = nil
	if record.Zone != nil {
		dnsZone := getDNSZone(*record.Zone)
		z = &dnsZone
	}
	return hdns.RecordUpdateOpts{
		Name:  record.Name,
		Zone:  z,
		Type:  hdns.RecordType(record.Type),
		Value: record.Value,
		Ttl:   getDNSTtl(record.TTL),
	}
}

func getPaginationMeta(meta hdns.Meta) *model.Pagination {
	if meta.Pagination == nil {
		return nil
	}
	pag := meta.Pagination
	return &model.Pagination{
		ItemsPerPage: pag.PerPage,
		PageIdx:      pag.Page,
		LastPage:     pag.LastPage,
		TotalCount:   pag.TotalEntries,
	}
}
