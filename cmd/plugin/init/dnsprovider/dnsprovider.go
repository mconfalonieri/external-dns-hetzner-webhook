package dnsprovider

import (
	"regexp"
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/configuration"
	"github.com/ionos-cloud/external-dns-ionos-plugin/internal/ionoscore"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/provider"
	log "github.com/sirupsen/logrus"
)

func Init(config configuration.Config) provider.Provider {
	var domainFilter endpoint.DomainFilter
	if config.RegexDomainFilter != "" {
		domainFilter = endpoint.NewRegexDomainFilter(
			regexp.MustCompile(config.RegexDomainFilter),
			regexp.MustCompile(config.RegexDomainExclusion),
		)
	} else {
		domainFilter = endpoint.NewDomainFilterWithExclusions(config.DomainFilter, config.ExcludeDomains)
	}

	log.Infof("Creating IONOS core provider with parameters Domain filter: %s , Exclude domain filter: %s, Regexp domain filter: %s, Regexp domain filter exclusion: %s, Dry run: %t",
		strings.Join(config.DomainFilter, ","),
		strings.Join(config.ExcludeDomains, ","),
		config.RegexDomainFilter,
		config.RegexDomainExclusion,
		config.DryRun)

	prov, err := ionoscore.NewProvider(domainFilter, config.DryRun)
	if err != nil {
		log.Fatalf("Failed to initialize IonosCore provider: %v", err)
	}

	return prov
}
