package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockStatus struct {
	ready  bool
	health bool
}

func (s *mockStatus) SetHealth(health bool) {
	s.health = health
}

func (s *mockStatus) SetReady(ready bool) {
	s.ready = ready
}

func Test_waitForSignal(t *testing.T) {
	name := "wait for signal test"
	actual := mockStatus{
		ready:  true,
		health: true,
	}
	expected := mockStatus{}
	bkpNotify := notify
	notify = func(sig chan os.Signal) {
		go func() {
			time.Sleep(time.Second)
			sig <- syscall.SIGTERM
		}()
	}

	t.Run(name, func(t *testing.T) {
		waitForSignal(&actual)
		assert.Equal(t, expected, actual)
	})

	notify = bkpNotify
}
