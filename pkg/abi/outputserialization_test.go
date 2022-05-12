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
	"context"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONSerializationFormatsTuple(t *testing.T) {

	abi := testABI(t, sampleABI1)
	assert.NotNil(t, abi)

	v, err := abi[0].Inputs.ParseJSON([]byte(`{
		"a": {
			"b": 12345,
			"c": [
				"abc",
				"def"
			],
			"d": "0xfeedbeef"
		}
	}`))
	assert.NoError(t, err)

	j1, err := v.JSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"a": {
			"b": "12345",
			"c": [
				"abc",
				"def"
			],
			"d": "feedbeef"
		}
	}`, string(j1))

	j2, err := NewSerializer().
		SetFormattingMode(FormatAsFlatArrays).
		SetIntSerializer(HexIntSerializer0xPrefix).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `[
		[
			"0x3039",
			[
				"abc",
				"def"
			],
			"0xfeedbeef"
		]
	]`, string(j2))

	j3, err := NewSerializer().
		SetFormattingMode(FormatAsSelfDescribingArrays).
		SetIntSerializer(func(i *big.Int) interface{} {
			return "0o" + i.Text(8)
		}).
		SetByteSerializer(func(b []byte) interface{} {
			return base64.StdEncoding.EncodeToString(b)
		}).
		SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `[
		{
			"name": "a",
			"type": "(uint256,string[2],bytes)",
			"value": [
				{
					"name": "b",
					"type": "uint256",
					"value": "0o30071"
				},				
				{
					"name": "c",
					"type": "string[2]",
					"value": [
						"abc",
						"def"
					]
				},
				{
					"name": "d",
					"type": "bytes",
					"value": "/u2+7w=="
				}
			]
		}	
	]`, string(j3))
}

func TestJSONSerializationForTypes(t *testing.T) {

	abi := testABI(t, sampleABI2)
	assert.NotNil(t, abi)

	v, err := abi[0].Inputs.ParseJSON([]byte(`{
		"a": 128,
		"b": -128,
		"c": "0xABCA79A8Ac11452F263A9861624c498220980Ca7",
		"d": true,
		"e": -1.28,
		"f": 1.28,
		"g": "0x09080706050403020100",
		"h": "0x",
		"i": "9c7e63a423cf0e0163fbab351d3833b4ba6f05faf7a6d199",
		"j": "Bob"
	}`))
	assert.NoError(t, err)

	j1, err := v.JSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"a": "128",
		"b": "-128",
		"c": "abca79a8ac11452f263a9861624c498220980ca7",
		"d": true,
		"e": "-1.28",
		"f": "1.28",
		"g": "09080706050403020100",
		"h": "",
		"i": "9c7e63a423cf0e0163fbab351d3833b4ba6f05faf7a6d199",
		"j": "Bob"
	}`, string(j1))

	i1, err := NewSerializer().SerializeInterface(v)
	assert.NoError(t, err)
	assert.Equal(t, "128", i1.(map[string]interface{})["a"])
}

func TestNumberIfPossibleSerialization(t *testing.T) {

	abi := testABI(t, `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint"
			},
			{
				"name": "b",
				"type": "int"
			},
			{
				"name": "c",
				"type": "ufixed"
			},
			{
				"name": "d",
				"type": "fixed"
			}
		  ],
		  "outputs": []
		}
	  ]`)
	assert.NotNil(t, abi)

	v, err := abi[0].Inputs.ParseJSON([]byte(`{
		"a": 9007199254740991,
		"b": -9007199254740991,
		"c": 9007199254740991,
		"d": -9007199254740991
	}`))
	assert.NoError(t, err)

	s := NewSerializer().
		SetFloatSerializer(NumberIfFitsOrBase10StringFloatSerializer)
	s.SetIntSerializer(NumberIfFitsOrBase10StringIntSerializer)

	j1, err := s.SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"a": 9007199254740991,
		"b": -9007199254740991,
		"c": 9007199254740991,
		"d": -9007199254740991
	}`, string(j1))

	v, err = abi[0].Inputs.ParseJSON([]byte(`{
		"a": 9007199254740992,
		"b": -9007199254740992,
		"c": 9007199254740992,
		"d": -9007199254740992
	}`))
	assert.NoError(t, err)

	j2, err := s.SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"a": "9007199254740992",
		"b": "-9007199254740992",
		"c": "9.007199255e+15",
		"d": "-9.007199255e+15"
	}`, string(j2))
}

func TestSerializeJSONBadComponent(t *testing.T) {

	_, err := NewSerializer().SerializeJSON(&ComponentValue{})
	assert.Regexp(t, "FF22041", err)

	_, err = NewSerializer().SerializeJSON(&ComponentValue{
		Component: &typeComponent{
			cType: 99,
		},
	})
	assert.Regexp(t, "FF22041", err)

	_, err = NewSerializer().serializeElementaryType(context.Background(), "", &ComponentValue{
		Component: &typeComponent{},
	})
	assert.Regexp(t, "FF22050", err)

	_, err = NewSerializer().SerializeJSON(&ComponentValue{
		Component: &typeComponent{
			cType: DynamicArrayComponent,
		},
		Children: []*ComponentValue{
			{},
		},
	})
	assert.Regexp(t, "FF22041", err)

	badTuple := &ComponentValue{
		Component: &typeComponent{
			cType: TupleComponent,
		},
		Children: []*ComponentValue{
			{Component: &typeComponent{keyName: "a", elementaryType: &elementaryTypeInfo{}}},
		},
	}
	_, err = NewSerializer().SerializeJSON(badTuple)
	assert.Regexp(t, "FF22050", err)
	_, err = NewSerializer().SetFormattingMode(FormatAsFlatArrays).SerializeJSON(badTuple)
	assert.Regexp(t, "FF22050", err)
	_, err = NewSerializer().SetFormattingMode(FormatAsSelfDescribingArrays).SerializeJSON(badTuple)
	assert.Regexp(t, "FF22050", err)
	_, err = NewSerializer().SetFormattingMode(999).SerializeJSON(badTuple)
	assert.Regexp(t, "FF22051", err)
}

func TestJSONSerializationFormatsAnonymousTuple(t *testing.T) {

	abi := testABI(t, `[
		{
		  "name": "foo",
		  "type": "function",
		  "inputs": [
			{
				"type": "address"
			},
			{
				"type": "uint"
			}
		  ],
		  "outputs": []
		}
	  ]`)
	assert.NotNil(t, abi)

	v, err := abi[0].Inputs.ParseJSON([]byte(`[
		"0x6c26465984ac94713E83300d1F002296772eBB64",
		1
	]`))
	assert.NoError(t, err)

	j1, err := v.JSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"0": "6c26465984ac94713e83300d1f002296772ebb64",
		"1": "1"
	}`, string(j1))

	j2, err := NewSerializer().
		SetFormattingMode(FormatAsFlatArrays).
		SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `[
		"6c26465984ac94713e83300d1f002296772ebb64",
		"1"
	]`, string(j2))

	j3, err := NewSerializer().
		SetFormattingMode(FormatAsSelfDescribingArrays).
		SerializeJSON(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `[
		{
			"name": "0",
			"type": "address",
			"value": "6c26465984ac94713e83300d1f002296772ebb64"
		},
		{
			"name": "1",
			"type": "uint256",
			"value": "1"
		}
	]`, string(j3))
}
