package dnsprovider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionoscloud"

	"github.com/caarlos0/env/v8"

	"github.com/ionos-cloud/external-dns-ionos-webhook/cmd/webhook/init/configuration"
	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionos"
	"github.com/ionos-cloud/external-dns-ionos-webhook/internal/ionoscore"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/provider"
	log "github.com/sirupsen/logrus"
)

const (
	webtokenIonosISSValue = "ionoscloud"
)

type IONOSProviderFactory func(domainFilter endpoint.DomainFilter, ionosConfig *ionos.Configuration) provider.Provider

func setDefaults(apiEndpointURL, authHeader string, ionosConfig *ionos.Configuration) {
	if ionosConfig.APIEndpointURL == "" {
		ionosConfig.APIEndpointURL = apiEndpointURL
	}
	if ionosConfig.AuthHeader == "" {
		ionosConfig.AuthHeader = authHeader
	}
}

var IonosCoreProviderFactory = func(domainFilter endpoint.DomainFilter, ionosConfig *ionos.Configuration) provider.Provider {
	setDefaults("https://api.hosting.ionos.com/dns", "X-API-Key", ionosConfig)
	return ionoscore.NewProvider(domainFilter, ionosConfig)
}

var IonosCloudProviderFactory = func(domainFilter endpoint.DomainFilter, ionosConfig *ionos.Configuration) provider.Provider {
	setDefaults("https://dns.de-fra.ionos.com", "Bearer", ionosConfig)
	return ionoscloud.NewProvider(domainFilter, ionosConfig)
}

func Init(config configuration.Config) (provider.Provider, error) {
	var domainFilter endpoint.DomainFilter
	createMsg := "Creating IONOS provider with "

	if config.RegexDomainFilter != "" {
		createMsg += fmt.Sprintf("Regexp domain filter: '%s', ", config.RegexDomainFilter)
		if config.RegexDomainExclusion != "" {
			createMsg += fmt.Sprintf("with exclusion: '%s', ", config.RegexDomainExclusion)
		}
		domainFilter = endpoint.NewRegexDomainFilter(
			regexp.MustCompile(config.RegexDomainFilter),
			regexp.MustCompile(config.RegexDomainExclusion),
		)
	} else {
		if config.DomainFilter != nil && len(config.DomainFilter) > 0 {
			createMsg += fmt.Sprintf("zoneNode filter: '%s', ", strings.Join(config.DomainFilter, ","))
		}
		if config.ExcludeDomains != nil && len(config.ExcludeDomains) > 0 {
			createMsg += fmt.Sprintf("Exclude domain filter: '%s', ", strings.Join(config.ExcludeDomains, ","))
		}
		domainFilter = endpoint.NewDomainFilterWithExclusions(config.DomainFilter, config.ExcludeDomains)
	}

	createMsg = strings.TrimSuffix(createMsg, ", ")
	if strings.HasSuffix(createMsg, "with ") {
		createMsg += "no kind of domain filters"
	}
	log.Info(createMsg)
	ionosConfig := ionos.Configuration{}
	if err := env.Parse(&ionosConfig); err != nil {
		return nil, fmt.Errorf("reading ionos ionosConfig failed: %v", err)
	}
	createProvider := detectProvider(&ionosConfig)
	provider := createProvider(domainFilter, &ionosConfig)
	return provider, nil
}

func detectProvider(ionosConfig *ionos.Configuration) IONOSProviderFactory {
	split := strings.Split(ionosConfig.APIKey, ".")
	if len(split) == 3 {
		tokenBytes, err := base64.RawStdEncoding.DecodeString(split[1])
		if err != nil {
			return IonosCoreProviderFactory
		}
		var tokenMap map[string]interface{}
		err = json.Unmarshal(tokenBytes, &tokenMap)
		if err != nil {
			return IonosCoreProviderFactory
		}
		if tokenMap["iss"] == webtokenIonosISSValue {
			return IonosCloudProviderFactory
		}
	}
	return IonosCoreProviderFactory
}
