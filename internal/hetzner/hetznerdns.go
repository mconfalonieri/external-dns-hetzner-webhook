package hetzner

import (
	"context"

	hdns "github.com/jobstoit/hetzner-dns-go/dns"
)

/*
type apiClient interface {
	GetZones(ctx context.Context, name string, searchName string, page int, perPage int) (*hdns.ZonesResponse, error)
	GetRecords(ctx context.Context, zone_id string, page int, perPage int) (*hdns.RecordsResponse, error)
	CreateRecord(ctx context.Context, record hdns.RecordRequest) (*hdns.RecordResponse, error)
	UpdateRecord(ctx context.Context, record hdns.RecordRequest) (*hdns.RecordResponse, error)
	DeleteRecord(ctx context.Context, recordId string) error
}
*/

// hetznerDNS is the DNS client API.
type hetznerDNS struct {
	client *hdns.Client
}

// NewHetznerDNS returns a new client.
func NewHetznerDNS(apiKey string) *hetznerDNS {
	return &hetznerDNS{
		client: hdns.NewClient(hdns.WithToken(apiKey)),
	}
}

// GetZones returns the available zones.
func (h hetznerDNS) GetZones(ctx context.Context, opts hdns.ZoneListOpts) ([]*hdns.Zone, *hdns.Response, error) {
	zoneClient := h.client.Zone
	return zoneClient.List(ctx, opts)
}

// GetRecords returns the records for a given zone.
func (h hetznerDNS) GetRecords(ctx context.Context, opts hdns.RecordListOpts,
) ([]*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.List(ctx, opts)
}

// CreateRecord creates a record.
func (h hetznerDNS) CreateRecord(ctx context.Context, opts hdns.RecordCreateOpts) (*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Create(ctx, opts)
}

// UpdateRecord updates a single record.
func (h hetznerDNS) UpdateRecord(ctx context.Context, record *hdns.Record, opts hdns.RecordUpdateOpts) (*hdns.Record, *hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Update(ctx, record, opts)
}

// DeleteRecord deletes a single record.
func (h hetznerDNS) DeleteRecord(ctx context.Context, record *hdns.Record) (*hdns.Response, error) {
	recordClient := h.client.Record
	return recordClient.Delete(ctx, record)
}
