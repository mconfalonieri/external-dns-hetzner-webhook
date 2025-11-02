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

	"sigs.k8s.io/external-dns/endpoint"
)

const (
	regex_1char = "^[a-z0-9A-Z]$"
	regex_label = "^[a-z0-9A-Z][a-z0-9A-Z_\\-./]*[a-z0-9A-Z]$"
	regex_value = "^[a-z0-9A-Z][a-z0-9A-Z_\\-.]*[a-z0-9A-Z]$"
)

var (
	regex1CharChecker = regexp.MustCompile(regex_1char)
	regexLabelChecker = regexp.MustCompile(regex_label)
	regexValueChecker = regexp.MustCompile(regex_value)
)

// formatLabels formats the labels back into a string.
func formatLabels(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	pairs := make([]string, 0)
	for k, v := range labels {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ";")
}

// checkLabel checks if the label is correct.
func checkLabel(label string) error {
	if label == "" {
		return errors.New("empty label is not acceptable")
	} else if !(len(label) == 1 && regex1CharChecker.MatchString(label)) && !(len(label) > 1 && regexLabelChecker.MatchString(label)) {
		return fmt.Errorf("label [%s] is not acceptable", label)
	}
	return nil
}

// checkValue checks if the value is correct.
func checkValue(value string) error {
	if value != "" && !(len(value) == 1 && regex1CharChecker.MatchString(value)) && !(len(value) > 1 && regexValueChecker.MatchString(value)) {
		return fmt.Errorf("value \"%s\" is not acceptable", value)
	}
	return nil
}

// parsePair parses the label=value pairs and does a formal checking on their
// syntax according to Hetzner formal requirements:
// https://docs.hetzner.cloud/reference/cloud#labels
func parsePair(item string) (string, string, error) {
	if item == "" {
		return "", "", fmt.Errorf("empty string provided")
	}
	rawPair := strings.Split(item, "=")
	if len(rawPair) != 2 {
		return "", "", fmt.Errorf("malformed pair \"%s\"", item)
	}
	label := rawPair[0]
	value := rawPair[1]

	// Check the label
	if err := checkLabel(label); err != nil {
		return "", "", fmt.Errorf("in pair \"%s\": %s", item, err.Error())
	}

	// Check the value
	if err := checkValue(value); err != nil {
		return "", "", fmt.Errorf("for label [%s]: %s", label, err.Error())
	}

	return label, value, nil
}

// extractLabelMap extracts the labels from a field.
func extractLabelMap(value string) (map[string]string, error) {
	items := strings.Split(value, ";")
	labels := make(map[string]string, len(items))
	for idx, item := range items {
		label, value, err := parsePair(item)
		if err != nil {
			return nil, fmt.Errorf("malformed pair in position %d: %s", idx, err.Error())
		}
		labels[label] = value
	}
	return labels, nil
}

// extractHetznerLabels extracts the label map if available. A map is always
// instantiated if there is no error.
func extractHetznerLabels(ps endpoint.ProviderSpecific) (map[string]string, error) {
	const LABELS = "hetzner-labels"
	for _, p := range ps {
		if p.Name == LABELS {
			labels, err := extractLabelMap(p.Value)
			if err != nil {
				return nil, fmt.Errorf("cannot process labels: %s", err.Error())
			}
			return labels, nil
		}
	}
	return map[string]string{}, nil
}
