/*
 * ZoneIDMap - A zone-ID mapper that uses int64 as type.
 *
 * This file is a MODIFIED version of ExternalDNS ZoneIDMap.
 *
 * Copyright 2023 Marco Confalonieri.
 * Copyright 2017 The Kubernetes Authors.
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

package hetznercloud

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/idna"
)

type zoneIDName map[int64]string

func (z zoneIDName) Add(zoneID int64, zoneName string) {
	z[zoneID] = zoneName
}

// FindZone identifies the most suitable DNS zone for a given hostname.
// It returns the zone ID and name that best match the hostname.
//
// The function processes the hostname by splitting it into labels and
// converting each label to its Unicode form using IDNA (Internationalized
// Domain Names for Applications) standards.
//
// Labels containing underscores ('_') are skipped during Unicode conversion.
// This is because underscores are often used in special DNS records (e.g.,
// SRV records as per RFC 2782, or TXT record for services) that are not
// IDNA-aware and cannot represent non-ASCII labels. Skipping these labels
// ensures compatibility with such use cases.
func (z zoneIDName) FindZone(hostname string) (int64, string) {
	var name string
	domainLabels := strings.Split(hostname, ".")
	for i, label := range domainLabels {
		if strings.Contains(label, "_") {
			continue
		}
		convertedLabel, err := idna.ToUnicode(label)
		if err != nil {
			log.Warnf("Failed to convert label %q of hostname %q to its Unicode form: %v", label, hostname, err)
			convertedLabel = label
		}
		domainLabels[i] = convertedLabel
	}
	name = strings.Join(domainLabels, ".")

	suitableZoneID := int64(-1)
	suitableZoneName := ""

	for zoneID, zoneName := range z {
		if name == zoneName || strings.HasSuffix(name, "."+zoneName) {
			if suitableZoneName == "" || len(zoneName) > len(suitableZoneName) {
				suitableZoneID = zoneID
				suitableZoneName = zoneName
			}
		}
	}
	return suitableZoneID, suitableZoneName
}
