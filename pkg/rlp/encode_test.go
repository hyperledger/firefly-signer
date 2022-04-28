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

func TestEncode(t *testing.T) {

	assert.Equal(t, []byte{}, int64ToMinimalBytes(0))

	assert.Equal(t, big.NewInt(0x7FFFFFFFFFFFFFF0).Bytes(), int64ToMinimalBytes(0x7FFFFFFFFFFFFFF0))

	assert.Equal(t, []byte{0x80}, ByteArray("").Encode())

	assert.Equal(t, []byte{0x0f}, ByteArray([]byte{0x0f}).Encode())

	assert.Equal(t, []byte{0x83, 'd', 'o', 'g'}, ByteArray("dog").Encode())

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
	}, ByteArray("Lorem ipsum dolor sit amet, consectetur adipisicing elit").Encode())

}
