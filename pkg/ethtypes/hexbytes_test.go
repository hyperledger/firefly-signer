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

package ethtypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexBytes(t *testing.T) {

	testStruct := struct {
		H1 HexBytesPlain    `json:"h1"`
		H2 HexBytesPlain    `json:"h2"`
		H3 HexBytes0xPrefix `json:"h3"`
		H4 HexBytes0xPrefix `json:"h4"`
	}{}

	testData := `{
		"h1": "0xabcd1234",
		"h2": "ffff0000",
		"h3": "0xFEEDBEEF",
		"h4": "9009a00e"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.NoError(t, err)

	assert.Equal(t, "abcd1234", testStruct.H1.String())
	assert.Equal(t, "ffff0000", testStruct.H2.String())
	assert.Equal(t, "0xfeedbeef", testStruct.H3.String())
	assert.Equal(t, "0x9009a00e", testStruct.H4.String())

	jsonSerialized, err := json.Marshal(&testStruct)
	assert.JSONEq(t, `{
		"h1": "abcd1234",
		"h2": "ffff0000",
		"h3": "0xfeedbeef",
		"h4": "0x9009a00e"
	}`, string(jsonSerialized))
}

func TestHexBytesFailNonHex(t *testing.T) {

	testStruct := struct {
		H1 HexBytesPlain `json:"h1"`
	}{}

	testData := `{
		"h1": "wrong"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "bad hex", err)
}

func TestHexBytesFailNonString(t *testing.T) {

	testStruct := struct {
		H1 HexBytesPlain `json:"h1"`
	}{}

	testData := `{
		"h1": {}
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Error(t, err)
}

func TestHexByteConstructors(t *testing.T) {
	assert.Equal(t, HexBytes0xPrefix{0x01, 0x02}, MustNewHexBytes0xPrefix("0x0102"))
	assert.Panics(t, func() {
		MustNewHexBytes0xPrefix("!wrong")
	})
}

func TestHexByteEqual(t *testing.T) {
	assert.True(t, HexBytesPlain(nil).Equals(nil))
	assert.False(t, HexBytesPlain(nil).Equals(HexBytesPlain{0x00}))
	assert.False(t, (HexBytesPlain{0x00}).Equals(nil))
	assert.True(t, (HexBytesPlain{0x00}).Equals(HexBytesPlain{0x00}))

	assert.True(t, HexBytes0xPrefix(nil).Equals(nil))
	assert.False(t, HexBytes0xPrefix(nil).Equals(HexBytes0xPrefix{0x00}))
	assert.False(t, (HexBytes0xPrefix{0x00}).Equals(nil))
	assert.True(t, (HexBytes0xPrefix{0x00}).Equals(HexBytes0xPrefix{0x00}))
}
