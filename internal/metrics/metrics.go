/*
 * Metrics - OpenMetrics implementation.
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
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// metrics instance
var metrics *OpenMetrics

type OpenMetrics struct {
	registry *prometheus.Registry

	successfulApiCallsTotal *prometheus.CounterVec
	failedApiCallsTotal     *prometheus.CounterVec

	filteredOutZones prometheus.Gauge
	apiDelayCount    *prometheus.HistogramVec
}

// GetOpenMetricsInstance returns the current OpenMetrics instance or creates a
// new one if required.
func GetOpenMetricsInstance() *OpenMetrics {
	if metrics == nil {
		reg := prometheus.NewRegistry()
		metrics = &OpenMetrics{
			registry: reg,
			successfulApiCallsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "successful_api_calls_total",
					Help: "The number of successful Hetzner API calls",
				},
				[]string{"action"},
			),
			failedApiCallsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "failed_api_calls_total",
					Help: "The number of Hetzner API calls that returned an error",
				},
				[]string{"action"},
			),
			filteredOutZones: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "filtered_out_zones",
				Help: "The number of zones excluded by the domain filter",
			}),
			apiDelayCount: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "api_delay_count",
					Help:    "Histogram of the delay in milliseconds when calling the Hetzner API",
					Buckets: []float64{10, 100, 250, 500, 1000, 1500, 2000},
				},
				[]string{"action"},
			),
		}
		reg.MustRegister(metrics.successfulApiCallsTotal)
		reg.MustRegister(metrics.failedApiCallsTotal)
		reg.MustRegister(metrics.filteredOutZones)
		reg.MustRegister(metrics.apiDelayCount)
	}
	return metrics
}

// getLabels builds the label map.
func getLabels(action string) prometheus.Labels {
	return prometheus.Labels{"action": action}
}

func (m OpenMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// IncSuccessfulApiCallsTotal increments the successful_api_calls_total counter.
func (m *OpenMetrics) IncSuccessfulApiCallsTotal(action string) {
	m.successfulApiCallsTotal.With(getLabels(action)).Inc()
}

// IncFailedApiCallsTotal increments the failed_api_calls_total counter.
func (m *OpenMetrics) IncFailedApiCallsTotal(action string) {
	m.failedApiCallsTotal.With(getLabels(action)).Inc()
}

// SetFilteredOutZones sets the value for the filtered_out_zones gauge.
func (m *OpenMetrics) SetFilteredOutZones(num int) {
	m.filteredOutZones.Set(float64(num))
}

func (m *OpenMetrics) AddApiDelayCount(action string, delay int64) {
	m.apiDelayCount.With(getLabels(action)).Observe(float64(delay))
}
