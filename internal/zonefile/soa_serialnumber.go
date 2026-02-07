/*
 * SOASerialNumber - SOA serial number manipulation.
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
package zonefile

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	// format of the date part of the serial number
	fmtSOADate = "20060102"
)

// SOASerialNumber represents a SOA serial number
type SOASerialNumber struct {
	date    string
	version int
}

// collectDate collects the date part from the serial number.
func collectDate(sn string) (string, error) {
	datePart := sn[:8]
	date, err := time.Parse(fmtSOADate, datePart)
	if err != nil {
		return "", fmt.Errorf("cannot parse date in serial number \"%s\"", sn)
	}
	nowDate, err := time.Parse(fmtSOADate, time.Now().Format(fmtSOADate))
	if err != nil {
		msg := fmt.Sprintf("Internal conversion went wrong: %s", err.Error())
		panic(msg)
	}
	if date.After(nowDate) {
		return "", fmt.Errorf("unexpected date part \"%s\" is in the future", datePart)
	}
	return datePart, nil
}

// NewSOASerialNumber creates a serial number from a string.
func NewSOASerialNumber(sn string) (*SOASerialNumber, error) {
	if len(sn) != 10 {
		return nil, fmt.Errorf("serial number \"%s\" is unsupported", sn)
	}
	datePart, err := collectDate(sn)
	if err != nil {
		return nil, err
	}
	version, err := strconv.Atoi(sn[8:])
	if err != nil {
		return nil, fmt.Errorf("cannot parse version in serial number \"%s\": %w", sn, err)
	}
	if version < 0 || version > 99 {
		return nil, fmt.Errorf("version %d is not supported", version)
	}
	return &SOASerialNumber{
		date:    datePart,
		version: version,
	}, nil
}

// CreateSOASerialNumber creates a new serial number for today.
func CreateSOASerialNumber() *SOASerialNumber {
	return &SOASerialNumber{
		date:    time.Now().Format(fmtSOADate),
		version: 0,
	}
}

// Inc increments the version number.
func (s *SOASerialNumber) Inc() error {
	nowDate := time.Now().Format(fmtSOADate)
	if nowDate != s.date {
		s.date = nowDate
		s.version = 0
		return nil
	}
	if s.version == 99 {
		return errors.New("cannot increment version as it is 99")
	}
	s.version++
	return nil
}

// String returns a string representation of the serial number.
func (s SOASerialNumber) String() string {
	return fmt.Sprintf("%s%02d", s.date, s.version)
}

// Uint32 returns a uint32 representation of the serial number.
func (s SOASerialNumber) Uint32() uint32 {
	str := s.String()
	n, err := strconv.Atoi(str)
	if err != nil {
		msg := fmt.Sprintf("wrong internal conversion on \"%s\": %v", str, err)
		panic(msg)
	}
	return uint32(n)
}
