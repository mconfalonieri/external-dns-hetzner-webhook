package provider

import (
	"testing"

	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/stretchr/testify/require"
)

func TestNewBaseProvider(t *testing.T) {
	mockDomainFilter := endpoint.NewDomainFilter([]string{"a.de"})
	baseProvider := NewBaseProvider(mockDomainFilter)

	require.Equal(t, *baseProvider, BaseProvider{mockDomainFilter})
}

func TestBaseProvider_GetDomainFilter(t *testing.T) {
	mockDomainFilter := endpoint.NewDomainFilter([]string{"a.de"})
	baseProvider := NewBaseProvider(mockDomainFilter)

	result := baseProvider.GetDomainFilter()

	require.Equal(t, mockDomainFilter, result)
}

func TestBaseProvider_AdjustEndpoints(t *testing.T) {
	// Create a BaseProvider instance with a domain filter.
	domainFilter := endpoint.NewDomainFilter([]string{"example.com"})
	baseProvider := NewBaseProvider(domainFilter)

	// Create some sample endpoints using the NewEndpoint function.
	endpoint1 := endpoint.NewEndpoint("example.com", "A", "1.2.3.4")
	endpoint2 := endpoint.NewEndpoint("sub.example.com", "CNAME", "example.com")
	endpoint3 := endpoint.NewEndpoint("example.com", "A", "5.6.7.8")

	// Create a slice of endpoints to adjust.
	endpoints := []*endpoint.Endpoint{endpoint1, endpoint2, endpoint3}

	// Call the AdjustEndpoints method to get adjusted endpoints.
	adjustedEndpoints := baseProvider.AdjustEndpoints(endpoints)

	// Assert that the adjustedEndpoints are the same as the input endpoints.
	require.Equal(t, endpoints, adjustedEndpoints)
}
