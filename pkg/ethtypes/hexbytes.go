// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ethtypes

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// HexBytesPlain is simple bytes that are JSON stored/retrieved as hex
type HexBytesPlain []byte

// HexBytes0xPrefix are serialized to JSON as hex with an `0x` prefix
type HexBytes0xPrefix []byte

func (h HexBytesPlain) Equals(h2 HexBytesPlain) bool {
	return bytes.Equal(h, h2)
}

func (h *HexBytesPlain) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*h, err = hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return fmt.Errorf("bad hex: %s", err)
	}
	return nil
}

func (h HexBytesPlain) String() string {
	return hex.EncodeToString(h)
}

func (h HexBytesPlain) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.String())), nil
}

func (h *HexBytes0xPrefix) UnmarshalJSON(b []byte) error {
	return ((*HexBytesPlain)(h)).UnmarshalJSON(b)
}

func (h HexBytes0xPrefix) Equals(h2 HexBytes0xPrefix) bool {
	return bytes.Equal(h, h2)
}

func (h HexBytes0xPrefix) String() string {
	return "0x" + hex.EncodeToString(h)
}

func (h HexBytes0xPrefix) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.String())), nil
}

func NewHexBytes0xPrefix(s string) (HexBytes0xPrefix, error) {
	h, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}
	return HexBytes0xPrefix(h), nil
}

func MustNewHexBytes0xPrefix(s string) HexBytes0xPrefix {
	h, err := NewHexBytes0xPrefix(s)
	if err != nil {
		panic(err)
	}
	return h
}
