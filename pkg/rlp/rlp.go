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

import "math/big"

type rlpOffset byte

const (
	/**
	 * [0x80] If a string is 0-55 bytes long, the RLP encoding consists of a single byte with value
	 * 0x80 plus the length of the string followed by the string. The range of the first byte is
	 * thus [0x80, 0xb7].
	 */
	shortString rlpOffset = 0x80

	/**
	 * [0xb7] If a string is more than 55 bytes long, the RLP encoding consists of a single byte
	 * with value 0xb7 plus the length of the length of the string in binary form, followed by the
	 * length of the string, followed by the string. For example, a length-1024 string would be
	 * encoded as \xb9\x04\x00 followed by the string. The range of the first byte is thus [0xb8,
	 * 0xbf].
	 */
	// longString rlpOffset = 0xb7

	/**
	 * [0xc0] If the total payload of a list (i.e. the combined length of all its items) is 0-55
	 * bytes long, the RLP encoding consists of a single byte with value 0xc0 plus the length of the
	 * list followed by the concatenation of the RLP encodings of the items. The range of the first
	 * byte is thus [0xc0, 0xf7].
	 */
	// shortList rlpOffset = 0xc0

	/**
	 * [0xf7] If the total payload of a list is more than 55 bytes long, the RLP encoding consists
	 * of a single byte with value 0xf7 plus the length of the length of the list in binary form,
	 * followed by the length of the list, followed by the concatenation of the RLP encodings of the
	 * items. The range of the first byte is thus [0xf8, 0xff].
	 */
	// longList rlpOffset = 0xf7
)

type ByteArray []byte

type Element interface {
	IsList() bool
	Value() []byte
	Children() List
}

type List []Element

func WrapString(s string) ByteArray {
	return ByteArray(s)
}

func WrapInt(i *big.Int) ByteArray {
	return ByteArray(i.Bytes())
}

func (r ByteArray) Int() *big.Int {
	if r == nil {
		return nil
	}
	i := new(big.Int)
	return i.SetBytes(r)
}

func (r ByteArray) Encode() []byte {
	return encodeBytes(r, shortString)
}
