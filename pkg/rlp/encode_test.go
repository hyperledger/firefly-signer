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

package rlp

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeData(t *testing.T) {

	d := Data{}
	assert.Nil(t, d.List())
	assert.False(t, d.IsList())

	assert.Equal(t, []byte{}, int64ToMinimalBytes(0))

	assert.Equal(t, big.NewInt(0x7FFFFFFFFFFFFFF0).Bytes(), int64ToMinimalBytes(0x7FFFFFFFFFFFFFF0))

	assert.Equal(t, []byte{0x80}, WrapString("").Encode())

	assert.Equal(t, []byte{0x0f}, Data([]byte{0x0f}).Encode())

	assert.Equal(t, []byte{0x83, 'd', 'o', 'g'}, WrapString("dog").Encode())

	assert.Equal(t, []byte{0x00}, Data{0x00}.Encode())

	assert.Equal(t, []byte{
		0xb8,
		0x38,
		'L',
		'o',
		'r',
		'e',
		'm',
		' ',
		'i',
		'p',
		's',
		'u',
		'm',
		' ',
		'd',
		'o',
		'l',
		'o',
		'r',
		' ',
		's',
		'i',
		't',
		' ',
		'a',
		'm',
		'e',
		't',
		',',
		' ',
		'c',
		'o',
		'n',
		's',
		'e',
		'c',
		't',
		'e',
		't',
		'u',
		'r',
		' ',
		'a',
		'd',
		'i',
		'p',
		'i',
		's',
		'i',
		'c',
		'i',
		'n',
		'g',
		' ',
		'e',
		'l',
		'i',
		't',
	}, WrapString("Lorem ipsum dolor sit amet, consectetur adipisicing elit").Encode())

	expected := make([]byte, 56)
	expected[0] = 0xb7
	assert.Equal(t, expected, make(Data, 55).Encode())
}

func TestEncodeIntegers(t *testing.T) {

	assert.Equal(t, []byte{0x0f}, WrapInt(big.NewInt(0x0f)).Encode())

	assert.Equal(t, []byte{0x82, 0x04, 0x00}, WrapInt(big.NewInt(0x400)).Encode())

	assert.Equal(t, []byte{0x80}, WrapInt(big.NewInt(0)).Encode())

	assert.Equal(t, int64(0xfeedbeef), Data{0xfe, 0xed, 0xbe, 0xef}.Int().Int64())

	assert.Nil(t, Data(nil).Int())

}

func TestEncodeList(t *testing.T) {

	l := List{}
	assert.Equal(t, l, l.List())
	assert.True(t, l.IsList())

	assert.Equal(t, []byte{0xc8, 0x83, 'c', 'a', 't', 0x83, 'd', 'o', 'g'},
		List{WrapString("cat"), WrapString("dog")}.Encode())

	assert.Equal(t, []byte{0xc0}, List{}.Encode())

	assert.Equal(t, []byte{
		0xc7,
		0xc0,
		0xc1,
		0xc0,
		0xc3,
		0xc0,
		0xc1,
		0xc0,
	}, List{
		List{},
		List{
			List{},
		},
		List{
			List{},
			List{
				List{},
			},
		},
	}.Encode())

	assert.Equal(t, []byte{
		0xc6,
		0x82,
		0x7a,
		0x77,
		0xc1,
		0x04,
		0x01,
	}, List{
		WrapString("zw"),
		List{
			WrapInt(big.NewInt(4)),
		},
		WrapInt(big.NewInt(1)),
	}.Encode())

}
