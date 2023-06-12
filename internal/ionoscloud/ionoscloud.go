package ionoscloud

import (
	"context"

	"github.com/ionos-cloud/external-dns-ionos-plugin/internal/ionos"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/plan"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/provider"
)

// DNSService interface with needed zone and records method
type DNSService interface {
	// TODO define neeeded methods: get zones, create record, delete record, update record(update record type)
}

type DNSClient struct {
	// client *sdk.APIClient
}

// Provider extends base provider to work with paas dns rest API
type Provider struct {
	provider.BaseProvider
	// client DNSClient

	// domainFilter endpoint.DomainFilter
	DryRun bool
}

// NewProvider returns an instance of new provider
func NewProvider(domainFilter endpoint.DomainFilter, configuration *ionos.Configuration, dryRun bool) (*Provider, error) {
	// TODO create client
	// TODO create provider

	return &Provider{}, nil
}

// TODO client methods interface and impelemtations
// get record
// create record
// update record/both versions
// delete record

// TODO record to endpoint / endpoint to record conversion

// TODO Records
// call get zones
// filter zones
// get records
func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	return nil, nil
}

// filter records
// convert records to endpoint

// TODO Apply changes
// filter change slices
// call methods for changes
func (p *Provider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	return nil
}
