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

func TestHexUint64Ok(t *testing.T) {

	testStruct := struct {
		I1 *HexUint64 `json:"i1"`
		I2 *HexUint64 `json:"i2"`
		I3 *HexUint64 `json:"i3"`
		I4 *HexUint64 `json:"i4"`
		I5 *HexUint64 `json:"i5,omitempty"`
	}{}

	testData := `{
		"i1": "0xabcd1234",
		"i2": "54321",
		"i3": 12345
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.NoError(t, err)

	assert.Equal(t, uint64(0xabcd1234), testStruct.I1.Uint64())
	assert.Equal(t, uint64(0xabcd1234), testStruct.I1.Uint64OrZero())
	assert.Equal(t, uint64(54321), testStruct.I2.Uint64())
	assert.Equal(t, uint64(12345), testStruct.I3.Uint64())
	assert.Nil(t, testStruct.I4)
	assert.Equal(t, uint64(0), testStruct.I4.Uint64OrZero()) // BigInt() safe on nils
	assert.Equal(t, "0x0", testStruct.I4.String())
	assert.Nil(t, testStruct.I5)
	assert.Equal(t, uint64(12345), testStruct.I3.Uint64())

	jsonSerialized, err := json.Marshal(&testStruct)
	assert.JSONEq(t, `{
		"i1": "0xabcd1234",
		"i2": "0xd431",
		"i3": "0x3039",
		"i4": null
	}`, string(jsonSerialized))

}

func TestHexUint64MissingBytes(t *testing.T) {

	testStruct := struct {
		I1 HexUint64 `json:"i1"`
	}{}

	testData := `{
		"i1": "0x"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "FF22088", err)
}

func TestHexUint64BadType(t *testing.T) {

	testStruct := struct {
		I1 HexUint64 `json:"i1"`
	}{}

	testData := `{
		"i1": {}
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "FF22091", err)
}

func TestHexUint64BadJSON(t *testing.T) {

	testStruct := struct {
		I1 HexUint64 `json:"i1"`
	}{}

	testData := `{
		"i1": null
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Error(t, err)
}

func TestHexUint64BadNegative(t *testing.T) {

	testStruct := struct {
		I1 HexUint64 `json:"i1"`
	}{}

	testData := `{
		"i1": "-12345"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "FF22090", err)
}

func TestHexUint64BadTooLarge(t *testing.T) {

	testStruct := struct {
		I1 HexUint64 `json:"i1"`
	}{}

	testData := `{
		"i1": "18446744073709551616"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "FF22090", err)
}

func TestHexUint64Constructor(t *testing.T) {
	assert.Equal(t, uint64(12345), HexUint64(12345).Uint64())
}

func TestScanUint64(t *testing.T) {
	var i HexUint64
	pI := &i
	err := pI.Scan(false)
	err = pI.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, "0x0", pI.String())
	err = pI.Scan(int64(5555))
	assert.Equal(t, "0x15b3", pI.String())
	assert.NoError(t, err)
	err = pI.Scan(uint64(9999))
	assert.Equal(t, "0x270f", pI.String())
	assert.NoError(t, err)
	err = pI.Scan(int64(-9999))
	assert.Regexp(t, "FF22092", err)
}
