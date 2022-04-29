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

func TestAddressCheckSum(t *testing.T) {

	testStruct := struct {
		Addr1 Address         `json:"addr1"`
		Addr2 Address         `json:"addr2"`
		Addr3 AddressPlainHex `json:"addr3"`
		Addr4 AddressPlainHex `json:"addr4"`
		Addr5 Address0xHex    `json:"addr5"`
		Addr6 Address0xHex    `json:"addr6"`
	}{}

	testData := `{
		"addr1": "0x3CCb85578722B5B9250C1a76b4967166a6Ff7B8b",
		"addr2": "162534E1aE19712499CE4CB05263D074D7F7aF90",
		"addr3": "0xEF15BBAB59891537E9FF75EB5E15D860D0E64117",
		"addr4": "A0361F594d5bb261Bc066458805d7aefFC4Ec94a",
		"addr5": "0xbD9E8378c52741943FCcDE9283db12aA8841a9F2",
		"addr6": "06942dc1fC868aF18132C0916dA3ae4ab58142a4"		
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.NoError(t, err)

	assert.Equal(t, "0x3CCb85578722B5B9250C1a76b4967166a6Ff7B8b", testStruct.Addr1.String())
	assert.Equal(t, "0x162534E1aE19712499CE4CB05263D074D7F7aF90", testStruct.Addr2.String())
	assert.Equal(t, "ef15bbab59891537e9ff75eb5e15d860d0e64117", testStruct.Addr3.String())
	assert.Equal(t, "a0361f594d5bb261bc066458805d7aeffc4ec94a", testStruct.Addr4.String())
	assert.Equal(t, "0xbd9e8378c52741943fccde9283db12aa8841a9f2", testStruct.Addr5.String())
	assert.Equal(t, "0x06942dc1fc868af18132c0916da3ae4ab58142a4", testStruct.Addr6.String())

	jsonSerialized, err := json.Marshal(&testStruct)
	assert.JSONEq(t, `{
		"addr1": "0x3CCb85578722B5B9250C1a76b4967166a6Ff7B8b",
		"addr2": "0x162534E1aE19712499CE4CB05263D074D7F7aF90",
		"addr3": "ef15bbab59891537e9ff75eb5e15d860d0e64117",
		"addr4": "a0361f594d5bb261bc066458805d7aeffc4ec94a",
		"addr5": "0xbd9e8378c52741943fccde9283db12aa8841a9f2",
		"addr6": "0x06942dc1fc868af18132c0916da3ae4ab58142a4"		
	}`, string(jsonSerialized))

}

func TestAddressFailLen(t *testing.T) {

	testStruct := struct {
		Addr1 Address `json:"addr1"`
	}{}

	testData := `{
		"addr1": "0x00"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "bad address - must be 20 bytes", err)
}

func TestAddressFailNonHex(t *testing.T) {

	testStruct := struct {
		Addr1 Address `json:"addr1"`
	}{}

	testData := `{
		"addr1": "wrong"
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Regexp(t, "bad address", err)
}

func TestAddressFailNonString(t *testing.T) {

	testStruct := struct {
		Addr1 Address `json:"addr1"`
	}{}

	testData := `{
		"addr1": {}
	}`

	err := json.Unmarshal([]byte(testData), &testStruct)
	assert.Error(t, err)
}
