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
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/sha3"
)

// EthAddress uses full 0x prefixed checksum address format
type EthAddress [20]byte

// EthAddressNoChecksumNo0xPrefix can parse the same, but formats as just flat hex (no prefix)
type EthAddressNoChecksumNo0xPrefix EthAddress

// HexBytes is simple bytes that are JSON stored/retrieved as bytes
type HexBytes []byte

func (a *EthAddress) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return fmt.Errorf("bad address: %s", err)
	}
	if len(b) != 20 {
		return fmt.Errorf("bad address - must be 20 bytes (len=%d)", len(b))
	}
	copy(a[0:20], b)
	return nil
}

func (a EthAddress) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a.String())), nil
}

func (a EthAddress) String() string {

	// EIP-55: Mixed-case checksum address encoding
	// https://eips.ethereum.org/EIPS/eip-55

	hexAddr := hex.EncodeToString(a[0:20])
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(hexAddr))
	hexHash := hex.EncodeToString(hash.Sum(nil))

	buff := strings.Builder{}
	buff.WriteString("0x")
	for i := 0; i < 40; i++ {
		hexHashDigit, _ := strconv.ParseInt(string([]byte{hexHash[i]}), 16, 64)
		if hexHashDigit >= 8 {
			buff.WriteRune(unicode.ToUpper(rune(hexAddr[i])))
		} else {
			buff.WriteRune(unicode.ToLower(rune(hexAddr[i])))
		}
	}
	return buff.String()
}

func (a *EthAddressNoChecksumNo0xPrefix) UnmarshalJSON(b []byte) error {
	return ((*EthAddress)(a)).UnmarshalJSON(b)
}

func (a EthAddressNoChecksumNo0xPrefix) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a.String())), nil
}

func (a EthAddressNoChecksumNo0xPrefix) String() string {
	return hex.EncodeToString(a[0:20])
}
