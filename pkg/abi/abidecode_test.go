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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExampleABIDecode1(t *testing.T) {

	f := &Entry{
		Name: "baz",
		Inputs: ParameterArray{
			{Type: "uint32"},
			{Type: "bool"},
		},
	}

	d, err := hex.DecodeString("cdcd77c0" +
		"0000000000000000000000000000000000000000000000000000000000000045" +
		"0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)

	cv, err := f.DecodeABIInputs(d)
	assert.NoError(t, err)

	assert.Equal(t, "69", cv.Children[0].Value.(*big.Int).String())
	assert.Equal(t, "1", cv.Children[1].Value.(*big.Int).String()) // bool true

}

func TestExampleABIDecode2(t *testing.T) {

	f := &Entry{
		Name: "bar",
		Inputs: ParameterArray{
			{Type: "bytes3[2]"},
		},
	}

	d, err := hex.DecodeString("fce353f6" +
		"6162630000000000000000000000000000000000000000000000000000000000" + // first fixed-length entry in static array
		"6465660000000000000000000000000000000000000000000000000000000000", // second fixed-length in static array
	)
	assert.NoError(t, err)

	cv, err := f.DecodeABIInputs(d)
	assert.NoError(t, err)

	assert.Equal(t, "abc", string(cv.Children[0].Children[0].Value.([]byte)))
	assert.Equal(t, "def", string(cv.Children[0].Children[1].Value.([]byte)))

}

func TestExampleABIDecode3(t *testing.T) {

	f := &Entry{
		Name: "sam",
		Inputs: ParameterArray{
			{Type: "bytes"},
			{Type: "bool"},
			{Type: "uint[]"},
		},
	}

	d, err := hex.DecodeString("a5643bf2" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000060" + // location of data for 1st param
		"0000000000000000000000000000000000000000000000000000000000000001" + // boolean true
		"00000000000000000000000000000000000000000000000000000000000000a0" + // location of data for 3rd param
		// 1st param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000004" + // length in bytes
		"6461766500000000000000000000000000000000000000000000000000000000" + // "dave" padded right
		// 3rd param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000003" + // length of array
		"0000000000000000000000000000000000000000000000000000000000000001" + // first value
		"0000000000000000000000000000000000000000000000000000000000000002" + // second value
		"0000000000000000000000000000000000000000000000000000000000000003", // third value
	)
	assert.NoError(t, err)

	cv, err := f.DecodeABIInputs(d)
	assert.NoError(t, err)

	assert.Equal(t, "dave", string(cv.Children[0].Value.([]byte)))
	assert.Equal(t, "1", cv.Children[1].Value.(*big.Int).String())
	assert.Len(t, cv.Children[2].Children, 3)
	assert.Equal(t, "1", cv.Children[2].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "2", cv.Children[2].Children[1].Value.(*big.Int).String())
	assert.Equal(t, "3", cv.Children[2].Children[2].Value.(*big.Int).String())

}

func TestExampleABIDecode4(t *testing.T) {

	f := &Entry{
		Name: "f",
		Inputs: ParameterArray{
			{Type: "uint"},
			{Type: "uint32[]"},
			{Type: "bytes10"},
			{Type: "bytes"},
		},
	}

	d, err := hex.DecodeString("8be65246" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000123" + // 0x123 padded to 32 b
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset of start of 2nd param
		"3132333435363738393000000000000000000000000000000000000000000000" + // 0x1234567890 padded to 32 b
		"00000000000000000000000000000000000000000000000000000000000000e0" + // offset of start of 4th param
		// 2nd param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000002" + // 2 elements in array
		"0000000000000000000000000000000000000000000000000000000000000456" + // first element
		"0000000000000000000000000000000000000000000000000000000000000789" + // second element
		// 4th param (dynamic)
		"000000000000000000000000000000000000000000000000000000000000000d" + // 13 bytes
		"48656c6c6f2c20776f726c642100000000000000000000000000000000000000", // the string
	)
	assert.NoError(t, err)

	cv, err := f.DecodeABIInputs(d)
	assert.NoError(t, err)

	assert.Equal(t, int64(0x123), cv.Children[0].Value.(*big.Int).Int64())
	assert.Len(t, cv.Children[1].Children, 2)
	assert.Equal(t, int64(0x456), cv.Children[1].Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(0x789), cv.Children[1].Children[1].Value.(*big.Int).Int64())
	assert.Equal(t, "1234567890", string(cv.Children[2].Value.([]byte)))
	assert.Equal(t, "Hello, world!", string(cv.Children[3].Value.([]byte)))

}

func TestExampleABIDecode5(t *testing.T) {

	f := &Entry{
		Name: "g",
		Inputs: ParameterArray{
			{Type: "uint[][]"},
			{Type: "string[]"},
		},
	}

	d, err := hex.DecodeString("2289b18c" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of [[1, 2], [3]]
		"0000000000000000000000000000000000000000000000000000000000000140" + // offset of ["one", "two", "three"]
		"0000000000000000000000000000000000000000000000000000000000000002" + // count for [[1, 2], [3]]
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of [1, 2]
		"00000000000000000000000000000000000000000000000000000000000000a0" + // offset of [3]
		"0000000000000000000000000000000000000000000000000000000000000002" + // count for [1, 2]
		"0000000000000000000000000000000000000000000000000000000000000001" + // encoding of 1
		"0000000000000000000000000000000000000000000000000000000000000002" + // encoding of 2
		"0000000000000000000000000000000000000000000000000000000000000001" + // count for [3]
		"0000000000000000000000000000000000000000000000000000000000000003" + // encoding of 3
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for ["one", "two", "three"]
		"0000000000000000000000000000000000000000000000000000000000000060" + // offset for "one"
		"00000000000000000000000000000000000000000000000000000000000000a0" + // offset for "two"
		"00000000000000000000000000000000000000000000000000000000000000e0" + // offset for "three"
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for "one"
		"6f6e650000000000000000000000000000000000000000000000000000000000" + // encoding of "one"
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for "two"
		"74776f0000000000000000000000000000000000000000000000000000000000" + // encoding of "two"
		"0000000000000000000000000000000000000000000000000000000000000005" + // count for "three"
		"7468726565000000000000000000000000000000000000000000000000000000", // encoding of "three"
	)
	assert.NoError(t, err)

	cv, err := f.DecodeABIInputs(d)
	assert.NoError(t, err)

	assert.Len(t, cv.Children[0].Children, 2)
	assert.Len(t, cv.Children[0].Children[0].Children, 2)
	assert.Equal(t, int64(1), cv.Children[0].Children[0].Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(2), cv.Children[0].Children[0].Children[1].Value.(*big.Int).Int64())
	assert.Len(t, cv.Children[1].Children, 3)
	assert.Equal(t, "one", cv.Children[1].Children[0].Value)
	assert.Equal(t, "two", cv.Children[1].Children[1].Value)
	assert.Equal(t, "three", cv.Children[1].Children[2].Value)

}
