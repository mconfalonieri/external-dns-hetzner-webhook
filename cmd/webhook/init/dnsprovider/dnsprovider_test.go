package dnsprovider

import (
	"testing"

	"external-dns-hetzner-webhook/internal/hetznercloud"

	"external-dns-hetzner-webhook/cmd/webhook/init/configuration"
	"external-dns-hetzner-webhook/internal/hetznercore"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	cases := []struct {
		name          string
		config        configuration.Config
		env           map[string]string
		providerType  string
		expectedError string
	}{
		{
			name:         "minimal config for hetzner core provider",
			config:       configuration.Config{},
			env:          map[string]string{"IONOS_API_KEY": "apikey must be there"},
			providerType: "core",
		},
		{
			name:   "minimal config for hetzner cloud provider ( token is jwt with payload iss=hetznercloud )",
			config: configuration.Config{},
			env: map[string]string{
				"IONOS_API_KEY": "algorithm.eyAiaXNzIiA6ICJpb25vc2Nsb3VkIiB9.signature",
			},
			providerType: "cloud",
		},
		{
			name:          "without api key you are not able to create provider",
			config:        configuration.Config{},
			expectedError: "reading hetzner hetznerConfig failed: env: environment variable \"IONOS_API_KEY\" should not be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			dnsProvider, err := Init(tc.config)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError, "expecting error")
				return
			}
			assert.NoErrorf(t, err, "error creating provider")
			assert.NotNil(t, dnsProvider)
			if tc.providerType == "core" {
				_, ok := dnsProvider.(*hetznercore.Provider)
				assert.True(t, ok, "provider is not of type hetznercore.Provider")
			} else if tc.providerType == "cloud" {
				_, ok := dnsProvider.(*hetznercloud.Provider)
				assert.True(t, ok, "provider is not of type hetznercloud.Provider")
			}
		})
	}
}
