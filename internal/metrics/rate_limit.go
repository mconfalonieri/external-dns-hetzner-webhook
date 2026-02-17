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
	fmtHeaderNotFound  = "header %s not found"
	fmtUnexpectedValue = "header %s had unexpected value \"%s\""

	rlLimit     = "Ratelimit-Limit"
	rlRemaining = "Ratelimit-Remaining"
	rlReset     = "Ratelimit-Reset"
)

// rateLimit holds the rate limit information.
type rateLimit struct {
	limit     int
	remaining int
	reset     uint64
}

// readLimit reads the rate limit.
func readLimit(h http.Header) (int, error) {
	strLimit := h.Get(rlLimit)
	if strLimit == "" {
		return 0, fmt.Errorf(fmtHeaderNotFound, rlLimit)
	}
	limit, err := strconv.Atoi(strLimit)
	if err != nil {
		return 0, fmt.Errorf(fmtUnexpectedValue, rlLimit, strLimit)
	}
	return limit, nil
}

// readRemaining reads the remaining rate limit.
func readRemaining(h http.Header) (int, error) {
	strRemaining := h.Get(rlRemaining)
	if strRemaining == "" {
		return 0, fmt.Errorf(fmtHeaderNotFound, rlRemaining)
	}
	remaining, err := strconv.Atoi(strRemaining)
	if err != nil {
		return 0, fmt.Errorf(fmtUnexpectedValue, rlRemaining, strRemaining)
	}
	return remaining, nil
}

// readReset reads the next rate limit reset.
func readReset(h http.Header) (uint64, error) {
	strReset := h.Get(rlReset)
	if strReset == "" {
		return 0, fmt.Errorf(fmtHeaderNotFound, rlReset)
	}
	reset, err := strconv.ParseUint(strReset, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(fmtUnexpectedValue, rlReset, strReset)
	}
	return reset, err
}

// parseRateLimit parses the rate limit information from a HTTP header and
// returns it, or raises an error otherwise.
func parseRateLimit(h http.Header) (*rateLimit, error) {
	limit, err := readLimit(h)
	if err != nil {
		return nil, err
	}
	remaining, err := readRemaining(h)
	if err != nil {
		return nil, err
	}
	reset, err := readReset(h)
	if err != nil {
		return nil, err
	}
	return &rateLimit{
		limit:     limit,
		remaining: remaining,
		reset:     reset,
	}, nil
}
