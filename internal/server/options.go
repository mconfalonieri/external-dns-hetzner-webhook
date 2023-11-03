/*
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
	"time"
)

// ServerOptions contains the argument passed as environment variables that
// influence the server.
type ServerOptions struct {
	// Webhook host
	WebhookHost string `env:"WEBHOOK_HOST" default:"localhost"`
	// Webhook port
	WebhookPort uint16 `env:"WEBHOOK_PORT" default:"8888"`
	// Readiness and liveness probe host
	HealthHost string `env:"HEALTH_HOST" default:"0.0.0.0"`
	// Readiness and liveness probe port
	HealthPort uint16 `env:"HEALTH_PORT" default:"8080"`
	// Read timeout in milliseconds
	ReadTimeout int `env:"READ_TIMEOUT" default:"60000"`
	// Write timeout in milliseconds
	WriteTimeout int `env:"WRITE_TIMEOUT" default:"60000"`
}

// GetWebhookAddress returns the webhook address as "host:port".
func (o ServerOptions) GetWebhookAddress() string {
	return fmt.Sprintf("%s:%d", o.WebhookHost, o.WebhookPort)
}

// GetHealthAddress returns the address of the liveness and readiness probe as
// "host:port".
func (o ServerOptions) GetHealthAddress() string {
	return fmt.Sprintf("%s:%d", o.HealthHost, o.HealthPort)
}

// GetReadTimeout returns the read timeout in milliseconds.
func (o ServerOptions) GetReadTimeout() time.Duration {
	return time.Duration(o.ReadTimeout) * time.Millisecond
}

// GetWriteTimeout returns the read timeout in milliseconds.
func (o ServerOptions) GetWriteTimeout() time.Duration {
	return time.Duration(o.WriteTimeout) * time.Millisecond
}
