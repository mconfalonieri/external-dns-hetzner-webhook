/*
 * Status - server status.
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

// Status contains the health and ready statuses for the webhook.
type Status struct {
	healthy mutexedBool
	ready   mutexedBool
}

// SetHealthy sets the health status.
func (s *Status) SetHealthy(v bool) {
	s.healthy.Set(v)
}

// SetReady sets the readiness status.
func (s *Status) SetReady(v bool) {
	s.ready.Set(v)
}

// IsHealthy returns the healthy flag.
func (s *Status) IsHealthy() bool {
	return s.healthy.Get()
}

// IsReady returns the readiness status.
func (s *Status) IsReady() bool {
	return s.ready.Get()
}
