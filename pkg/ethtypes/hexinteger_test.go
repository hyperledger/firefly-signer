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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexIntegerOk(t *testing.T) {

	testStruct := struct {
		I1 *HexInteger `json:"i1"`
		I2 *HexInteger `json:"i2"`
		I3 *HexInteger `json:"i3"`
		I4 *HexInteger `json:"i4"`
		I5 *HexInteger `json:"i5,omitempty"`
	}{}

	testData := `{
		"i1": "0xabcd1234",
		"i2": "54321",
		"i3": 12345
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.NoError(t, err)

	assert.Equal(t, int64(0xabcd1234), testStruct.I1.BigInt().Int64())
	assert.Equal(t, int64(54321), testStruct.I2.BigInt().Int64())
	assert.Equal(t, int64(12345), testStruct.I3.BigInt().Int64())
	assert.Nil(t, testStruct.I4)
	assert.Equal(t, int64(0), testStruct.I4.BigInt().Int64()) // BigInt() safe on nils
	assert.Nil(t, testStruct.I5)
	assert.Equal(t, int64(12345), testStruct.I3.Int64())
	assert.Equal(t, uint64(12345), testStruct.I3.Uint64())

	jsonSerialized, err := json.Marshal(&testStruct)
	assert.JSONEq(t, `{
		"i1": "0xabcd1234",
		"i2": "0xd431",
		"i3": "0x3039",
		"i4": null
	}`, string(jsonSerialized))

}

func TestHexIntegerMissingBytes(t *testing.T) {

	testStruct := struct {
		I1 HexInteger `json:"i1"`
	}{}

	testData := `{
		"i1": "0x"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "unable to parse integer", err)
}

func TestHexIntegerBadType(t *testing.T) {

	testStruct := struct {
		I1 HexInteger `json:"i1"`
	}{}

	testData := `{
		"i1": {}
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "unable to parse integer", err)
}

func TestHexIntegerBadJSON(t *testing.T) {

	testStruct := struct {
		I1 HexInteger `json:"i1"`
	}{}

	testData := `{
		"i1": null
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Error(t, err)
}

func TestHexIntegerBadNegative(t *testing.T) {

	testStruct := struct {
		I1 HexInteger `json:"i1"`
	}{}

	testData := `{
		"i1": "-12345"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "negative values are not supported", err)
}

func TestHexIntConstructors(t *testing.T) {
	assert.Equal(t, int64(12345), NewHexInteger64(12345).BigInt().Int64())
	assert.Equal(t, int64(12345), NewHexInteger(big.NewInt(12345)).BigInt().Int64())
	assert.Equal(t, "0x0", NewHexInteger(big.NewInt(0)).String())
	assert.Equal(t, "0x1", NewHexInteger(big.NewInt(1)).String())
	assert.Equal(t, "0x1", NewHexIntegerU64(1).String())
}

func TestScan(t *testing.T) {
	i := &HexInteger{}
	err := i.Scan(false)
	err = i.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, "0x0", i.String())
	i.Scan(int64(5555))
	assert.Equal(t, "0x15b3", i.String())
	i.Scan(uint64(9999))
	assert.Equal(t, "0x270f", i.String())
}
