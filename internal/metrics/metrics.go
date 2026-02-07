/*
 * Metrics - OpenMetrics implementation.
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
	"github.com/prometheus/client_golang/prometheus"
)

// metrics instance
var metrics *OpenMetrics

// OpenMetrics is the instance that holds all the metrics infromation.
type OpenMetrics struct {
	registry *prometheus.Registry

	successfulApiCallsTotal *prometheus.CounterVec
	failedApiCallsTotal     *prometheus.CounterVec

	filteredOutZones prometheus.Gauge
	skippedRecords   *prometheus.GaugeVec
	apiDelayHist     *prometheus.HistogramVec
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
			skippedRecords: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "skipped_records",
					Help: "The number of skipped records per domain",
				},
				[]string{"zone"},
			),
			apiDelayHist: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "api_delay_hist",
					Help:    "Histogram of the delay in milliseconds when calling the Hetzner API",
					Buckets: []float64{10, 100, 250, 500, 1000, 1500, 2000},
				},
				[]string{"action"},
			),
		}
		reg.MustRegister(metrics.successfulApiCallsTotal)
		reg.MustRegister(metrics.failedApiCallsTotal)
		reg.MustRegister(metrics.filteredOutZones)
		reg.MustRegister(metrics.skippedRecords)
		reg.MustRegister(metrics.apiDelayHist)
	}
	return metrics
}

// GetRegistry returns the current prometheus registry.
func (m OpenMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// IncSuccessfulApiCallsTotal increments the successful_api_calls_total counter.
func (m *OpenMetrics) IncSuccessfulApiCallsTotal(action string) {
	label := prometheus.Labels{"action": action}
	m.successfulApiCallsTotal.With(label).Inc()
}

// IncFailedApiCallsTotal increments the failed_api_calls_total counter.
func (m *OpenMetrics) IncFailedApiCallsTotal(action string) {
	label := prometheus.Labels{"action": action}
	m.failedApiCallsTotal.With(label).Inc()
}

// SetFilteredOutZones sets the value for the filtered_out_zones gauge.
func (m *OpenMetrics) SetFilteredOutZones(num int) {
	m.filteredOutZones.Set(float64(num))
}

// SetSkippedRecords sets the value for the skipped_records gauge.
func (m *OpenMetrics) SetSkippedRecords(zone string, num int) {
	label := prometheus.Labels{"zone": zone}
	m.skippedRecords.With(label).Set(float64(num))
}

// AddApiDelayHist adds a value to the api_delay_hist histogram.
func (m *OpenMetrics) AddApiDelayHist(action string, delay int64) {
	label := prometheus.Labels{"action": action}
	m.apiDelayHist.With(label).Observe(float64(delay))
}
