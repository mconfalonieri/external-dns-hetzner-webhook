package dnsprovider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/caarlos0/env/v8"

	"external-dns-hetzner-webhook/cmd/webhook/init/configuration"

	"external-dns-hetzner-webhook/internal/hetzner"
	"external-dns-hetzner-webhook/pkg/endpoint"
	"external-dns-hetzner-webhook/pkg/provider"

	log "github.com/sirupsen/logrus"
)

const (
	authHeader = "Auth-API-Token"
	apiURL     = "https://api.hosting.hetzner.com/dns"
	apiVersion = "v1"
)

type HetznerProviderFactory func(baseProvider *provider.BaseProvider, hetznerConfig *hetzner.Configuration) provider.Provider

func setDefaults(apiEndpointURL, authHeader string, hetznerConfig *hetzner.Configuration) {
	if hetznerConfig.APIEndpointURL == "" {
		hetznerConfig.APIEndpointURL = apiEndpointURL
	}
	if hetznerConfig.AuthHeader == "" {
		hetznerConfig.AuthHeader = authHeader
	}
}

var HetznerDNSProviderFactory = func(baseProvider *provider.BaseProvider, hetznerConfig *hetzner.Configuration) provider.Provider {
	setDefaults("https://api.hosting.hetzner.com/dns", "Auth-API-Token", hetznerConfig)
	return hetznercore.NewProvider(baseProvider, hetznerConfig)
}

var HetznerCloudProviderFactory = func(baseProvider *provider.BaseProvider, hetznerConfig *hetzner.Configuration) provider.Provider {
	setDefaults("https://dns.de-fra.hetzner.com", "Bearer", hetznerConfig)
	return hetznercloud.NewProvider(baseProvider, hetznerConfig)
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
	hetznerConfig := hetzner.Configuration{}
	if err := env.Parse(&hetznerConfig); err != nil {
		return nil, fmt.Errorf("reading hetzner hetznerConfig failed: %v", err)
	}
	createProvider := detectProvider(&hetznerConfig)
	baseProvider := provider.NewBaseProvider(domainFilter)
	hetznerProvider := createProvider(baseProvider, &hetznerConfig)
	return hetznerProvider, nil
}

func detectProvider(hetznerConfig *hetzner.Configuration) IONOSProviderFactory {
	split := strings.Split(hetznerConfig.APIKey, ".")
	if len(split) == 3 {
		tokenBytes, err := base64.RawStdEncoding.DecodeString(split[1])
		if err != nil {
			return HetznerCoreProviderFactory
		}
		var tokenMap map[string]interface{}
		err = json.Unmarshal(tokenBytes, &tokenMap)
		if err != nil {
			return HetznerCoreProviderFactory
		}
		if tokenMap["iss"] == webtokenHetznerISSValue {
			return HetznerCloudProviderFactory
		}
	}
	return HetznerCoreProviderFactory
}
