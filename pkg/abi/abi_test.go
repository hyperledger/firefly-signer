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
	"encoding/json"
	"math/big"
	"testing"

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
					"type": "string[]"
				}
			]
		}
	  ],
	  "outputs": []
	}
  ]`

func TestABIGetTupleTypeTree(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)

	assert.Equal(t, "foo((uint256,string[]))", abi[0].String())
	tc, err := abi[0].Inputs[0].TypeComponentTree()
	assert.NoError(t, err)

	assert.Equal(t, TupleComponent, tc.ComponentType())
	assert.Len(t, tc.TupleChildren(), 2)
	assert.Equal(t, "(uint256,string[])", tc.String())

	assert.Equal(t, ElementaryComponent, tc.TupleChildren()[0].ComponentType())
	assert.Equal(t, ElementaryTypeUint, tc.TupleChildren()[0].ElementaryType())

	assert.Equal(t, VariableArrayComponent, tc.TupleChildren()[1].ComponentType())
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
	assert.Regexp(t, "FF00161", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF00161", err)

	err = abi[0].Inputs[0].Validate()
	assert.Regexp(t, "FF00161", err)

	assert.Empty(t, abi[0].Inputs[0].String())

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
	assert.Regexp(t, "FF00161", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF00161", err)

	err = abi[0].Outputs[0].Validate()
	assert.Regexp(t, "FF00161", err)

	assert.Empty(t, abi[0].Outputs[0].String())

}

func TestABIParseObjectOk(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)
	inputs := abi[0].Inputs

	values := `{
		"a": {
			"b": 12345,
			"c": ["string1", "string2"]
		}
	}`
	var jv interface{}
	err = json.Unmarshal([]byte(values), &jv)
	assert.NoError(t, err)

	cv, err := inputs.ParseExternalData(jv)
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)

}

func TestABIParseArrayOk(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)
	inputs := abi[0].Inputs

	values := `[
		[
			12345,
			["string1", "string2"]
		]
	]`

	cv, err := inputs.ParseExternalJSON([]byte(values))
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)

}

func TestABIParseMissingRoot(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)
	inputs := abi[0].Inputs

	values := `{}`

	_, err = inputs.ParseExternalJSON([]byte(values))
	assert.Regexp(t, "FF00173", err)

}
