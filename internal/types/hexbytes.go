// Copyright © 2022 Kaleido, Inc.
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

package types

import (
	// ISC licensed
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// HexBytesNoPrefix is simple bytes that are JSON stored/retrieved as hex
type HexBytesNoPrefix []byte

// HexBytes0xPrefix is simple bytes that can o
type HexBytes0xPrefix []byte

func (h *HexBytesNoPrefix) UnmarshalJSON(b []byte) error {
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

func (h HexBytesNoPrefix) String() string {
	return hex.EncodeToString(h)
}

func (h HexBytesNoPrefix) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.String())), nil
}

func (h *HexBytes0xPrefix) UnmarshalJSON(b []byte) error {
	return ((*HexBytesNoPrefix)(h)).UnmarshalJSON(b)
}

func (h HexBytes0xPrefix) String() string {
	return "0x" + hex.EncodeToString(h)
}

func (h HexBytes0xPrefix) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.String())), nil
}
