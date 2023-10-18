package hetzner

import (
	"regexp"

	"sigs.k8s.io/external-dns/endpoint"
)

// Configuration contains the Hetzner provider's configuration.
type Configuration struct {
	APIKey               string   `env:"HETZNER_API_KEY,notEmpty"`
	DryRun               bool     `env:"DRY_RUN" envDefault:"false"`
	Debug                bool     `env:"HETZNER_DEBUG" envDefault:"false"`
	BatchSize            int      `env:"BATCH_SIZE" envDefault:"100"`
	DefaultTTL           int      `env:"DEFAULT_TTL" envDefault:"7200"`
	DomainFilter         []string `env:"DOMAIN_FILTER" envDefault:""`
	ExcludeDomains       []string `env:"EXCLUDE_DOMAIN_FILTER" envDefault:""`
	RegexDomainFilter    string   `env:"REGEXP_DOMAIN_FILTER" envDefault:""`
	RegexDomainExclusion string   `env:"REGEXP_DOMAIN_FILTER_EXCLUSION" envDefault:""`
}

// GetDomainFilter returns the domain filter from the configuration.
func (config *Configuration) GetDomainFilter() endpoint.DomainFilter {
	var domainFilter endpoint.DomainFilter
	if config.RegexDomainFilter != "" {
		if config.RegexDomainExclusion != "" {
			domainFilter = endpoint.NewRegexDomainFilter(
				regexp.MustCompile(config.RegexDomainFilter),
				regexp.MustCompile(config.RegexDomainExclusion),
			)
		} else {
			domainFilter = endpoint.NewDomainFilterWithExclusions(config.DomainFilter, config.ExcludeDomains)
		}
	}
	return domainFilter
}
