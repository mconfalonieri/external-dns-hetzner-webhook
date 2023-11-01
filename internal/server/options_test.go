package server

import (
	"testing"
	"time"

	"github.com/codingconcepts/env"
	"gotest.tools/assert"
)

func Test_ServerOptions_defaults(t *testing.T) {
	s := ServerOptions{}
	if err := env.Set(&s); err != nil {
		t.Fail()
	}
	assert.DeepEqual(t, s.WebhookHost, "localhost")
	assert.DeepEqual(t, s.WebhookPort, 8888)
	assert.DeepEqual(t, s.HealthHost, "0.0.0.0")
	assert.DeepEqual(t, s.WebhookHost, 8080)
	assert.DeepEqual(t, s.ReadTimeout, 60000)
	assert.DeepEqual(t, s.WriteTimeout, 60000)
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

	assert.DeepEqual(t, wa, testWebhookAddress)
	assert.DeepEqual(t, ha, testHealthAddress)
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

	assert.DeepEqual(t, r, testReadTimeout)
	assert.DeepEqual(t, w, testWriteTimeout)
}
