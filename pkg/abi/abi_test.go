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

package abi

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

const sampleABI1 = `[
	{
	  "name": "foo",
	  "type": "function",
	  "inputs": [
		{
			"name": "a",
			"type": "tuple",
			"components": [
				{
					"name": "b",
					"type": "uint"
				},
				{
					"name": "c",
					"type": "string[2]"
				},
				{
					"name": "d",
					"type": "bytes"
				}
			]
		}
	  ],
	  "outputs": []
	}
  ]`

const sampleABI2 = `[
	{
	  "name": "foo",
	  "type": "function",
	  "inputs": [
		{
			"name": "a",
			"type": "uint8"
		},
		{
			"name": "b",
			"type": "int"
		},
		{
			"name": "c",
			"type": "address"
		},
		{
			"name": "d",
			"type": "bool"
		},
		{
			"name": "e",
			"type": "fixed64x10"
		},
		{
			"name": "f",
			"type": "ufixed"
		},
		{
			"name": "g",
			"type": "bytes10"
		},
		{
			"name": "h",
			"type": "bytes"
		},
		{
			"name": "i",
			"type": "function"
		},
		{
			"name": "j",
			"type": "string"
		}
	  ],
	  "outputs": []
	}
  ]`

func testABI(t *testing.T, abiJSON string) (abi ABI) {
	err := json.Unmarshal([]byte(abiJSON), &abi)
	assert.NoError(t, err)
	return abi
}

func TestABIGetTupleTypeTree(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)

	assert.Equal(t, "foo((uint256,string[2],bytes))", abi[0].String())
	tc, err := abi[0].Inputs[0].TypeComponentTree()
	assert.NoError(t, err)

	assert.Equal(t, TupleComponent, tc.ComponentType())
	assert.Len(t, tc.TupleChildren(), 3)
	assert.Equal(t, "(uint256,string[2],bytes)", tc.String())

	assert.Equal(t, ElementaryComponent, tc.TupleChildren()[0].ComponentType())
	assert.Equal(t, ElementaryTypeUint, tc.TupleChildren()[0].ElementaryType())

	assert.Equal(t, FixedArrayComponent, tc.TupleChildren()[1].ComponentType())
	assert.Equal(t, ElementaryComponent, tc.TupleChildren()[1].ArrayChild().ComponentType())
	assert.Equal(t, ElementaryTypeString, tc.TupleChildren()[1].ArrayChild().ElementaryType())

}

func TestABIModifyReParse(t *testing.T) {

	abiString := `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	assert.Equal(t, "foo(uint256)", abi[0].String())

	// Just changing doesn't help, as it's cached
	abi[0].Inputs[0].Type = "uint128"
	assert.Equal(t, "foo(uint256)", abi[0].String())

	// Re-parse sorts it
	abi.Validate()
	assert.Equal(t, "foo(uint128)", abi[0].String())
	assert.Equal(t, "c56cb6b0", abi[0].ID())

}

func TestABIModifyBadInputs(t *testing.T) {

	abiString := `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint-1"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	assert.Empty(t, abi[0].String())

	err = abi.Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Inputs[0].Validate()
	assert.Regexp(t, "FF22028", err)

	assert.Empty(t, abi[0].Inputs[0].String())
	assert.Empty(t, abi[0].ID())

}

func TestABIModifyBadOutputs(t *testing.T) {

	abiString := `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [],
		  "outputs": [
			  {
				"name": "a",
				"type": "uint-1"
			  }
		  ]
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	err = abi.Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Outputs[0].Validate()
	assert.Regexp(t, "FF22028", err)

	assert.Empty(t, abi[0].Outputs[0].String())

}

func TestParseExternalJSONObjectModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `{
		"a": {
			"b": 12345,
			"c": ["string1", "string2"],
			"d": "0xfeedbeef"
		}
	}`
	var jv interface{}
	err := json.Unmarshal([]byte(values), &jv)
	assert.NoError(t, err)

	cv, err := inputs.ParseExternalData(jv)
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseExternalJSONArrayModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `[
		[
			12345,
			["string1", "string2"],
			"0xfeedbeef"
		]
	]`

	cv, err := inputs.ParseExternalJSON([]byte(values))
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseExternalJSONMixedModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `[
		{
			"b": 12345,
			"c": ["string1", "string2"],
			"d": "feedbeef"
		}
	]`

	cv, err := inputs.ParseExternalJSON([]byte(values))
	assert.NoError(t, err)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestABIParseCoerceGoTypes(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := map[interface{}]interface{}{
		"a": map[interface{}]interface{}{
			TestStringCustomType("b"): TestInt32CustomType(12345),
			&TestStringable{"c"}: []*TestStringable{
				{"string1"},
				{"string2"},
			},
			"d": TestByteArrayCustomType{0xfe, 0xed, 0xbe, 0xef},
		},
	}

	cv, err := inputs.ParseExternalData(values)
	assert.NoError(t, err)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseExternalJSONArrayLotsOfTypes(t *testing.T) {

	inputs := testABI(t, sampleABI2)[0].Inputs

	values := `[
		"-12345",
		"0x12345",
		"0x4a0d852eBb58FC88Cb260Bb270AE240f72EdC45B",
		true,
		"-1.2345",
		1.2345,
		"0xfeedbeef",
		"00010203040506070809",
		"00",
		"test string"
	]`

	cv, err := inputs.ParseExternalJSON([]byte(values))
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, int64(-12345), cv.Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(0x12345), cv.Children[1].Value.(*big.Int).Int64())
	addrBytes, err := hex.DecodeString("4a0d852ebb58fc88cb260bb270ae240f72edc45b")
	assert.NoError(t, err)
	addrUint := new(big.Int).SetBytes(addrBytes)
	assert.Equal(t, addrUint.String(), cv.Children[2].Value.(*big.Int).String())
	assert.Equal(t, "1", cv.Children[3].Value.(*big.Int).String())
	assert.Equal(t, "-1.2345", cv.Children[4].Value.(*big.Float).String())
	assert.Equal(t, "1.2345", cv.Children[5].Value.(*big.Float).String())
	assert.Equal(t, "0xfeedbeef", ethtypes.HexBytes0xPrefix(cv.Children[6].Value.([]byte)).String())
	assert.Equal(t, "0x00010203040506070809", ethtypes.HexBytes0xPrefix(cv.Children[7].Value.([]byte)).String())
	assert.Equal(t, "0x00", ethtypes.HexBytes0xPrefix(cv.Children[8].Value.([]byte)).String())
	assert.Equal(t, "test string", cv.Children[9].Value)

}

func TestParseExternalJSONBadData(t *testing.T) {
	inputs := testABI(t, sampleABI1)[0].Inputs
	_, err := inputs.ParseExternalJSON([]byte(`{`))
	assert.Regexp(t, "unexpected end", err)

}

func TestParseExternalJSONBadABI(t *testing.T) {
	inputs := testABI(t, `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "wrong"
			}
		  ],
		  "outputs": []
		}
	  ]`)[0].Inputs
	_, err := inputs.ParseExternalJSON([]byte(`{}`))
	assert.Regexp(t, "FF22025", err)

}
