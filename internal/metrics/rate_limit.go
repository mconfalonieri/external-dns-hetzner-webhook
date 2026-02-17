/*
 * Rate limit - Rate limit headers
 *
 * Copyright 2026 Marco Confalonieri.
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
package metrics

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	rlLimit     = "RateLimit-Limit"
	rlRemaining = "RateLimit-Remaining"
	rlReset     = "RateLimit-Reset"
)

// rateLimit contains the rate limits
type rateLimit struct {
	limit     int
	remaining int
	reset     uint64
}

// parseRateLimits returns the rate limits.
func parseRateLimits(h http.Header) (*rateLimit, error) {
	strLimit := h.Get(rlLimit)
	if strLimit == "" {
		return nil, fmt.Errorf("header %s not found", rlLimit)
	}
	strRemaining := h.Get(rlRemaining)
	if strRemaining == "" {
		return nil, fmt.Errorf("header %s not found", rlRemaining)
	}
	strReset := h.Get(rlReset)
	if strReset == "" {
		return nil, fmt.Errorf("header %s not found", rlReset)
	}
	limit, err := strconv.Atoi(strLimit)
	if err != nil {
		return nil, err
	}
	remaining, err := strconv.Atoi(strRemaining)
	if err != nil {
		return nil, err
	}
	reset, err := strconv.ParseUint(strReset, 10, 64)
	if err != nil {
		return nil, err
	}
	return &rateLimit{
		limit:     limit,
		remaining: remaining,
		reset:     reset,
	}, nil
}
