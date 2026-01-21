/*
 * Labels - functions for handling provider-specific labels.
 *
 * Copyright 2024 Marco Confalonieri.
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
package hetznercloud

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

const (
	slashDefault   string = "--slash--"
	providerPrefix string = "webhook/hetzner-label-"

	regex_1char string = "^[a-z0-9A-Z]$"
	regex_label string = "^[a-z0-9A-Z][a-z0-9A-Z_\\-./]*[a-z0-9A-Z]$"
	regex_value string = "^[a-z0-9A-Z][a-z0-9A-Z_\\-.]*[a-z0-9A-Z]$"
)

var (
	regex1CharChecker = regexp.MustCompile(regex_1char)
	regexLabelChecker = regexp.MustCompile(regex_label)
	regexValueChecker = regexp.MustCompile(regex_value)
)

// formatLabels formats the labels into a printable string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	pairs := make([]string, 0)
	for k, v := range labels {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ";")
}

// getProviderSpecific returns an endpoint.ProviderSpecific object from a label
// map.
func getProviderSpecific(slash string, labels map[string]string) endpoint.ProviderSpecific {
	if len(labels) == 0 {
		log.Debug("No labels found")
		return nil
	}
	ps := make(endpoint.ProviderSpecific, 0)
	for label, value := range labels {
		label = strings.ReplaceAll(label, "/", slash)
		name := providerPrefix + label
		log.Debugf("Adding provider-specific: [%s: %s]", name, value)
		ps = append(ps, endpoint.ProviderSpecificProperty{
			Name:  name,
			Value: value,
		})
	}
	return ps
}

// checkLabel checks if the label is correct.
func checkLabel(label string) error {
	if label == "" {
		return errors.New("empty label is not acceptable")
	} else if !(len(label) == 1 && regex1CharChecker.MatchString(label)) && !(len(label) > 1 && regexLabelChecker.MatchString(label)) {
		return fmt.Errorf("label [%s] is not acceptable", label)
	} else if len(label) > 63 {
		return fmt.Errorf("label [%s...] is longer than 63 characters", label[:20])
	}
	return nil
}

// checkValue checks if the value is correct.
func checkValue(value string) error {
	if value != "" && !(len(value) == 1 && regex1CharChecker.MatchString(value)) && !(len(value) > 1 && regexValueChecker.MatchString(value)) {
		return fmt.Errorf("value \"%s\" is not acceptable", value)
	} else if len(value) > 63 {
		return fmt.Errorf("value \"%s...\" is longer than 63 characters", value[:20])
	}
	return nil
}

// extractHetznerLabels extracts the label map if available. A map is always
// instantiated if there is no error.
func extractHetznerLabels(slash string, ps endpoint.ProviderSpecific) (map[string]string, error) {
	if slash == "" {
		slash = slashDefault
	}
	labels := make(map[string]string, 0)
	for _, p := range ps {
		if strings.HasPrefix(p.Name, providerPrefix) {
			log.Debugf("Processing provider-specific: [%s: %s]", p.Name, p.Value)
			label := strings.TrimPrefix(p.Name, providerPrefix)
			label = strings.ReplaceAll(label, slash, "/")
			value := p.Value
			if err := checkLabel(label); err != nil {
				return nil, fmt.Errorf("cannot process label for [%s: \"%s\"]: %w", label, value, err)
			} else if err := checkValue(value); err != nil {
				return nil, fmt.Errorf("cannot process value for [%s: \"%s\"]: %w", label, value, err)
			}
			labels[label] = value
		} else {
			log.Debugf("Ignoring provider-specific: [%s: %s]", p.Name, p.Value)
		}
	}
	return labels, nil
}
