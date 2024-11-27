/*
 * Options - socket configurable options.
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
package server

import (
	"fmt"
	"os"
	"time"

	"github.com/codingconcepts/env"
	log "github.com/sirupsen/logrus"
)

const (
	// Deprecated socket environment variable for metrics host
	envDeprecatedMetricsHost = "HEALTH_HOST"
	// Deprecated socket environment variable for metrics port
	envDeprecatedMetricsPort = "HEALTH_PORT"
)

// SocketOptions contains the argument passed as environment variables that
// influence the socket configuration.
type SocketOptions struct {
	// Webhook host
	WebhookHost string `env:"WEBHOOK_HOST" default:"localhost"`
	// Webhook port
	WebhookPort uint16 `env:"WEBHOOK_PORT" default:"8888"`
	// Readiness and liveness probe host
	MetricsHost string `env:"METRICS_HOST" default:"0.0.0.0"`
	// Readiness and liveness probe port
	MetricsPort uint16 `env:"METRICS_PORT" default:"8080"`
	// Read timeout in milliseconds
	ReadTimeout int `env:"READ_TIMEOUT" default:"60000"`
	// Write timeout in milliseconds
	WriteTimeout int `env:"WRITE_TIMEOUT" default:"60000"`
}

// NewSocketOptions returns a pointer to a new SocketOptions instance. This
// instance will be populated with values taken from the relevant environment
// variables and a warning will be issued if a deprecated environment variable
// is used. If there is an issue while reading the environment variables
// an error is raised.
func NewSocketOptions() (*SocketOptions, error) {
	opt := &SocketOptions{}

	// Populate with values from environment.
	if err := env.Set(opt); err != nil {
		return nil, err
	}

	// Check if HEALTH_HOST is used
	if metricsHost := os.Getenv(envDeprecatedMetricsHost); metricsHost != "" {
		log.Warnf("Setting metrics hostname using the deprecated name %s", envDeprecatedMetricsHost)
		opt.MetricsHost = metricsHost
	}

	// Check if HEALTH_PORT is used
	if metricsPort := os.Getenv(envDeprecatedMetricsPort); metricsPort != "" {
		log.Warnf("Setting metrics port using the deprecated name %s", envDeprecatedMetricsPort)
		opt.MetricsHost = metricsPort
	}

	return opt, nil
}

// GetWebhookAddress returns the webhook socket address.
func (o SocketOptions) GetWebhookAddress() string {
	return fmt.Sprintf("%s:%d", o.WebhookHost, o.WebhookPort)
}

// GetHealthAddress returns the metrics socket address.
func (o SocketOptions) GetMetricsAddress() string {
	return fmt.Sprintf("%s:%d", o.MetricsHost, o.MetricsPort)
}

// GetReadTimeout returns the read timeout in milliseconds.
func (o SocketOptions) GetReadTimeout() time.Duration {
	return time.Duration(o.ReadTimeout) * time.Millisecond
}

// GetWriteTimeout returns the read timeout in milliseconds.
func (o SocketOptions) GetWriteTimeout() time.Duration {
	return time.Duration(o.WriteTimeout) * time.Millisecond
}
