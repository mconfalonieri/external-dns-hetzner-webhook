package hetzner

import (
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
