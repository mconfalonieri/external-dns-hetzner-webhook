package hetzner

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

// Configuration contains the Hetzner provider's configuration.
type Configuration struct {
	APIKey               string   `env:"HETZNER_API_KEY" required:"true"`
	DryRun               bool     `env:"DRY_RUN" default:"false"`
	Debug                bool     `env:"HETZNER_DEBUG" default:"false"`
	BatchSize            int      `env:"BATCH_SIZE" default:"100"`
	DefaultTTL           int      `env:"DEFAULT_TTL" default:"7200"`
	DomainFilter         []string `env:"DOMAIN_FILTER" default:""`
	ExcludeDomains       []string `env:"EXCLUDE_DOMAIN_FILTER" default:""`
	RegexDomainFilter    string   `env:"REGEXP_DOMAIN_FILTER" default:""`
	RegexDomainExclusion string   `env:"REGEXP_DOMAIN_FILTER_EXCLUSION" default:""`
}

// GetDomainFilter returns the domain filter from the configuration.
func (config *Configuration) GetDomainFilter() endpoint.DomainFilter {
	var domainFilter endpoint.DomainFilter
	createMsg := "Creating Hetzner provider with "

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
	return domainFilter
}
