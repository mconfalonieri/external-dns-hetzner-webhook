/*
 * Metrics - Unit tests.
 *
 * Copyright 2026 Marco Confalonieri.
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
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	testAction = "test_action"
	testZone   = "alpha.com"
)

func Test_GetOpenMetricsInstance(t *testing.T) {
	type testCase struct {
		name    string
		metrics *OpenMetrics
	}

	run := func(t *testing.T, tc testCase) {
		actual := GetOpenMetricsInstance()
		if tc.metrics != nil {
			assert.EqualValues(t, metrics, actual)
		} else {
			assert.NotNil(t, metrics)
		}
	}

	testCases := []testCase{
		{
			name:    "new instance required",
			metrics: nil,
		},
		{
			name:    "existing instance",
			metrics: &OpenMetrics{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_OpenMetrics_IncSuccessfulApiCallsTotal(t *testing.T) {
	metrics = nil
	expected := float64(1)

	GetOpenMetricsInstance().IncSuccessfulApiCallsTotal(testAction)
	actual := testutil.ToFloat64(metrics.successfulApiCallsTotal)

	assert.Equal(t, expected, actual)
}

func Test_OpenMetrics_IncFailedApiCallsTotal(t *testing.T) {
	metrics = nil
	expected := float64(1)

	GetOpenMetricsInstance().IncFailedApiCallsTotal(testAction)
	actual := testutil.ToFloat64(metrics.failedApiCallsTotal)

	assert.Equal(t, expected, actual)
}

func Test_OpenMetrics_SetFilteredOutZones(t *testing.T) {
	metrics = nil
	const val = 5
	expected := float64(val)

	GetOpenMetricsInstance().SetFilteredOutZones(val)
	actual := testutil.ToFloat64(metrics.filteredOutZones)

	assert.Equal(t, expected, actual)
}

func Test_OpenMetrics_SetSkippedRecords(t *testing.T) {
	metrics = nil
	const val = 5
	expected := float64(val)

	GetOpenMetricsInstance().SetSkippedRecords(testZone, val)
	actual := testutil.ToFloat64(metrics.skippedRecords)

	assert.Equal(t, expected, actual)
}
