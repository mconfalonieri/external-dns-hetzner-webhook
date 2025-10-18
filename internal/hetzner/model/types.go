/*
 * API-independent types.
 *
 * Copyright 2025 Marco Confalonieri.
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

package model

import "time"

// Zone represents a DNS zone.
type Zone struct {
	ID       string
	Created  time.Time
	Modified time.Time
	Name     string
	TTL      int
}

// Record represents a DNS record.
type Record struct {
	ID       string
	Created  time.Time
	Modified time.Time
	Zone     *Zone
	Type     string
	Name     string
	Value    string
	TTL      int
}

// ListOpts contains the common options for paged results.
type ListOpts struct {
	PageIdx      int
	ItemsPerPage int
}

// RecordListOpts contains the options for record listing.
type RecordListOpts struct {
	ListOpts
	ZoneID string
}

// ZoneListOpts contains the options for Zone listing.
type ZoneListOpts struct {
	ListOpts
	SearchName string
	Name       string
}

// Pagination represents paginated information.
type Pagination struct {
	LastPage     int
	PageIdx      int
	ItemsPerPage int
	TotalCount   int
}
