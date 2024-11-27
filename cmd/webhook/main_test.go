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
