package main

import (
	"testing"

	"github.com/codingconcepts/env"
	"gotest.tools/assert"
)

func Test_ServerOptions_empty(t *testing.T) {
	s := serverOptions{}
	if err := env.Set(&s); err != nil {
		t.Fail()
	}
	assert.DeepEqual(t, s.Hostname, "0.0.0.0")
}

func Test_ServerOptions_value(t *testing.T) {
	const testAddress = "127.0.0.1"
	t.Setenv("SERVER_HOST", testAddress)
	s := serverOptions{}
	if err := env.Set(&s); err != nil {
		t.Fail()
	}
	assert.DeepEqual(t, s.Hostname, testAddress)
}
