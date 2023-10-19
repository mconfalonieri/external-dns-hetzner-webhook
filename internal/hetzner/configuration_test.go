package hetzner

import (
	"regexp"
	"testing"

	"gotest.tools/assert"
	"sigs.k8s.io/external-dns/endpoint"
)

func Test_GetDomainFilter(t *testing.T) {
	t.Setenv("HETZNER_API_KEY", "test-key")
	type testCase struct {
		name     string
		config   Configuration
		expected endpoint.DomainFilter
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.config.GetDomainFilter()
		actualJSON, _ := actual.MarshalJSON()
		expectedJSON, _ := tc.expected.MarshalJSON()
		assert.DeepEqual(t, actualJSON, expectedJSON)
	}

	testCases := []testCase{
		{
			name:     "No domain filters",
			config:   Configuration{},
			expected: endpoint.DomainFilter{},
		},
		{
			name: "Simple domain filter",
			config: Configuration{
				DomainFilter: []string{"example.com"},
			},
			expected: endpoint.NewDomainFilter([]string{"example.com"}),
		},
		{
			name: "Exclusion domain filter",
			config: Configuration{
				ExcludeDomains: []string{"example.com"},
			},
			expected: endpoint.NewDomainFilterWithExclusions(nil, []string{"example.com"}),
		},
		{
			name: "Both domain filters",
			config: Configuration{
				DomainFilter:   []string{"example-included.com"},
				ExcludeDomains: []string{"example-excluded.com"},
			},
			expected: endpoint.NewDomainFilterWithExclusions(
				[]string{"example-included.com"},
				[]string{"example-excluded.com"},
			),
		},
		{
			name: "Regular expression domain filters",
			config: Configuration{
				RegexDomainFilter:    `example-[a-z]+\.com`,
				RegexDomainExclusion: `[a-z]+-excluded\.com`,
			},
			expected: endpoint.NewRegexDomainFilter(
				regexp.MustCompile(`example-[a-z]+\.com`),
				regexp.MustCompile(`[a-z]+-excluded\.com`),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
