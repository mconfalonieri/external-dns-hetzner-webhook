package dnsprovider

import (
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/configuration"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	cases := []struct {
		name   string
		config configuration.Config
		env    map[string]string
	}{
		{
			name:   "minimal working config",
			config: configuration.Config{},
			env:    map[string]string{"IONOS_API_KEY": "apikey must be there"},
		},
	}

	// run test cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			dnsProvider := Init(tc.config)
			assert.NotNil(t, dnsProvider)
		})
	}
}
