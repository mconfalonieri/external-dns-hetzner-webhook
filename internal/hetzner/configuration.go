/*
 * Configuration - provider configuration
 *
 * Copyright 2023 Marco Confalonieri.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package hetzner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codingconcepts/env"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

// Configuration contains the Hetzner provider's configuration.
type Configuration struct {
	// Use the new Cloud API for DNS
	UseCloudAPI bool `env:"USE_CLOUD_API" default:"false"`
	// DNS API key or Cloud API key
	APIKey string `env:"HETZNER_API_KEY" required:"true"`
	// If true, do not execute actions on the API
	DryRun bool `env:"DRY_RUN" default:"false"`
	// Enable debugging logs
	Debug bool `env:"HETZNER_DEBUG" default:"false"`
	// Default batch size (max 100)
	BatchSize int `env:"BATCH_SIZE" default:"100"`
	// Default TTL when not specified
	DefaultTTL int `env:"DEFAULT_TTL" default:"7200"`
	// Domain filter
	DomainFilter []string `env:"DOMAIN_FILTER" default:""`
	// Excluded domains
	ExcludeDomains []string `env:"EXCLUDE_DOMAIN_FILTER" default:""`
	// Regular expression for domain filter
	RegexDomainFilter string `env:"REGEXP_DOMAIN_FILTER" default:""`
	// Regular expression for excluding domains
	RegexDomainExclusion string `env:"REGEXP_DOMAIN_FILTER_EXCLUSION" default:""`
}

// NewConfiguration creates a new configuration object.
func NewConfiguration() (*Configuration, error) {
	cfg := &Configuration{}

	// Populate with values from environment.
	if err := env.Set(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetDomainFilter returns the domain filter from the configuration. If the
// regular expression filters are set, the others are ignored.
func GetDomainFilter(config Configuration) *endpoint.DomainFilter {
	var domainFilter *endpoint.DomainFilter
	createMsg := "Creating Hetzner provider with "

	if config.RegexDomainFilter != "" {
		createMsg += fmt.Sprintf("regexp domain filter: '%s', ", config.RegexDomainFilter)
		if config.RegexDomainExclusion != "" {
			createMsg += fmt.Sprintf("with exclusion: '%s', ", config.RegexDomainExclusion)
		}
		domainFilter = endpoint.NewRegexDomainFilter(
			regexp.MustCompile(config.RegexDomainFilter),
			regexp.MustCompile(config.RegexDomainExclusion),
		)
	} else {
		if len(config.DomainFilter) > 0 {
			createMsg += fmt.Sprintf("zoneNode filter: '%s', ", strings.Join(config.DomainFilter, ","))
		}
		if len(config.ExcludeDomains) > 0 {
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
