/*
 * Changes Internals - unit tests
 *
 * Copyright 2024 Marco Confalonieri.
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
package provider

import (
	"external-dns-hetzner-webhook/internal/hetzner/model"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type changeType interface {
	GetLogFields() log.Fields
}

var defaultTTL = -1

func Test_GetLogFields(t *testing.T) {
	type testCase struct {
		name     string
		object   changeType
		expected log.Fields
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.object.GetLogFields()
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{
			name: "hetznerChangeCreate",
			object: hetznerChangeCreate{
				Name:  "testName",
				TTL:   defaultTTL,
				Value: "testValue",
				Type:  "CNAME",
				Zone: &model.Zone{
					ID:   "testZoneID",
					Name: "testZoneName",
				},
			},
			expected: log.Fields{
				"domain":     "testZoneName",
				"zoneID":     "testZoneID",
				"dnsName":    "testName",
				"recordType": "CNAME",
				"value":      "testValue",
				"ttl":        defaultTTL,
			},
		},
		{
			name: "hetznerChangeUpdate",
			object: &hetznerChangeUpdate{
				ID:    "recordID",
				Name:  "testNewName",
				TTL:   defaultTTL,
				Value: "testNewValue",
				Type:  "CNAME",
				Zone: &model.Zone{
					ID:   "testZoneID",
					Name: "testZoneName",
				},
			},
			expected: log.Fields{
				"domain":      "testZoneName",
				"zoneID":      "testZoneID",
				"recordID":    "recordID",
				"*dnsName":    "testNewName",
				"*recordType": "CNAME",
				"*value":      "testNewValue",
				"*ttl":        defaultTTL,
			},
		},
		{
			name: "hetznerChangeDelete",
			object: &hetznerChangeDelete{
				ID: "recordID",
				Zone: &model.Zone{
					ID:   "testZoneID",
					Name: "testZoneName",
				},
				Type:  "CNAME",
				Name:  "recordName",
				Value: "recordValue",
			},
			expected: log.Fields{
				"domain":     "testZoneName",
				"zoneID":     "testZoneID",
				"dnsName":    "recordName",
				"recordType": "CNAME",
				"value":      "recordValue",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
