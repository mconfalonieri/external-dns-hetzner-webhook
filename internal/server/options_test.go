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
	"testing"
	"time"

	"github.com/codingconcepts/env"
	"github.com/stretchr/testify/assert"
)

func Test_ServerOptions_defaults(t *testing.T) {
	s := ServerOptions{}
	if err := env.Set(&s); err != nil {
		t.Fail()
	}
	assert.Equal(t, s.WebhookHost, "localhost")
	assert.Equal(t, s.WebhookPort, uint16(8888))
	assert.Equal(t, s.HealthHost, "0.0.0.0")
	assert.Equal(t, s.HealthPort, uint16(8080))
	assert.Equal(t, s.ReadTimeout, 60000)
	assert.Equal(t, s.WriteTimeout, 60000)
}

func Test_ServerOptions_addresses(t *testing.T) {
	const testWebhookAddress = "10.0.0.1:1000"
	const testHealthAddress = "10.0.0.2:2000"
	s := ServerOptions{
		WebhookHost: "10.0.0.1",
		WebhookPort: 1000,
		HealthHost:  "10.0.0.2",
		HealthPort:  2000,
	}

	wa := s.GetWebhookAddress()
	ha := s.GetHealthAddress()

	assert.Equal(t, wa, testWebhookAddress)
	assert.Equal(t, ha, testHealthAddress)
}

func Test_ServerOptions_timeouts(t *testing.T) {
	const testReadTimeout = time.Duration(5000) * time.Millisecond
	const testWriteTimeout = time.Duration(15000) * time.Millisecond
	s := ServerOptions{
		ReadTimeout:  5000,
		WriteTimeout: 15000,
	}

	r := s.GetReadTimeout()
	w := s.GetWriteTimeout()

	assert.Equal(t, r, testReadTimeout)
	assert.Equal(t, w, testWriteTimeout)
}
