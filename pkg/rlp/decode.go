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

import "fmt"

const (
	maxUint32 = int64(^uint32(0))
	maxInt32  = int64(int32(maxUint32 >> 1))

	/**
	 * [0x80] If a string is 0-55 bytes long, the RLP encoding consists of a single byte with value
	 * 0x80 plus the length of the string followed by the string. The range of the first byte is
	 * thus [0x80, 0xb7].
	 */
	shortString byte = 0x80

	/**
	 * [0xb7] If a string is more than 55 bytes long, the RLP encoding consists of a single byte
	 * with value 0xb7 plus the length of the length of the string in binary form, followed by the
	 * length of the string, followed by the string. For example, a length-1024 string would be
	 * encoded as \xb9\x04\x00 followed by the string. The range of the first byte is thus [0xb8,
	 * 0xbf].
	 */
	longString byte = 0xb7

	/**
	 * [0xc0] If the total payload of a list (i.e. the combined length of all its items) is 0-55
	 * bytes long, the RLP encoding consists of a single byte with value 0xc0 plus the length of the
	 * list followed by the concatenation of the RLP encodings of the items. The range of the first
	 * byte is thus [0xc0, 0xf7].
	 */
	shortList byte = 0xc0

	/**
	 * [0xf7] If the total payload of a list is more than 55 bytes long, the RLP encoding consists
	 * of a single byte with value 0xf7 plus the length of the length of the list in binary form,
	 * followed by the length of the list, followed by the concatenation of the RLP encodings of the
	 * items. The range of the first byte is thus [0xf8, 0xff].
	 */
	longList byte = 0xf7

	/**
	 * [0x37] == (longList-shortList) == (longString-shortString)
	 * which means we can add it to either short offset, to get the long offset
	 */
	shortToLong byte = 0x37
)

// Decode will decode an RLP element at the beginning of the byte slice.
// An error is returned if problems are found in the RLP encoding
//
// The position of the first byte after the RLP element is returned.
// This will be the length of the byte slice, if the RLP element filled the
// entire slice. Otherwise, the position can be used to find/decode
// additional data (RLP or otherwise) in the byte slice.
//
// The element returned will be one of:
// - nil if passed an empty byte array
// - Data if the RLP stream contains a data element in the first position
// - List if the RLP stream contains a list in the first position
func Decode(rlpData []byte) (Element, int, error) {
	decoded, endPos, err := decode(rlpData, 1)
	if err != nil {
		return nil, -1, err
	}
	if len(decoded) >= 1 {
		return decoded[0], endPos, nil
	}
	return nil, 0, nil
}

func decode(rlpData []byte, limit int) (List, int, error) {
	l := List{}
	if len(rlpData) == 0 {
		return l, 0, nil
	}
	elements := 0
	pos := 0
	for pos < len(rlpData) && (limit < 0 || elements < limit) {

		prefix := rlpData[pos] & 0xff

		switch {
		case prefix < shortString:

			// 1. the data is a string if the range of the
			// first byte(i.e. prefix) is [0x00, 0x7f],
			// and the string is the first byte itself exactly;
			l = append(l, Data{rlpData[pos]})
			pos++

		case prefix == shortString:

			// null
			l = append(l, Data{})
			pos++

		case prefix > shortString && prefix <= longString:

			// 3. the data is a string if the range of the
			// first byte is [0xb8, 0xbf], and the length of the
			// string which length in bytes is equal to the
			// first byte minus 0xb7 follows the first byte,
			// and the string follows the length of the string;

			strLen := int(prefix - shortString)
			pos++
			if strLen > len(rlpData)-pos {
				return nil, -1, fmt.Errorf("length mismatch in RLP for short data (pos=%d len=%d)", pos, strLen)
			}
			d := make(Data, strLen)
			copy(d, rlpData[pos:pos+strLen])
			l = append(l, d)
			pos += strLen

		case prefix > longString && prefix < shortList:

			// 3. the data is a string if the range of the
			// first byte is [0xb8, 0xbf], and the length of the
			// string which length in bytes is equal to the
			// first byte minus 0xb7 follows the first byte,
			// and the string follows the length of the string;

			strLen, newPos, err := extractLongLen(false, prefix, pos, rlpData)
			if err != nil {
				return nil, -1, err
			}
			pos = newPos
			d := make(Data, strLen)
			copy(d, rlpData[pos:pos+strLen])
			l = append(l, d)
			pos += strLen

		case prefix >= shortList && prefix <= longList:

			// 4. the data is a list if the range of the
			// first byte is [0xc0, 0xf7], and the concatenation of
			// the RLP encodings of all items of the list which the
			// total payload is equal to the first byte minus 0xc0 follows the first byte;

			listLen := int(prefix - shortList)
			pos++
			if listLen > len(rlpData)-pos {
				return nil, -1, fmt.Errorf("length mismatch in RLP for short list (pos=%d len=%d)", pos, listLen)
			}
			child, _, err := decode(rlpData[pos:pos+listLen], -1)
			if err != nil {
				return nil, -1, err
			}
			l = append(l, child)
			pos += listLen

		case prefix > longList:

			// 5. the data is a list if the range of the
			// first byte is [0xf8, 0xff], and the total payload of the
			// list which length is equal to the
			// first byte minus 0xf7 follows the first byte,
			// and the concatenation of the RLP encodings of all items of
			// the list follows the total payload of the list;

			listLen, newPos, err := extractLongLen(true, prefix, pos, rlpData)
			if err != nil {
				return nil, -1, err
			}
			pos = newPos
			child, _, err := decode(rlpData[pos:pos+listLen], -1)
			if err != nil {
				return nil, -1, err
			}
			l = append(l, child)
			pos += listLen

		}

		elements++

	}

	return l, pos, nil
}

func extractLongLen(isList bool, prefixByte byte, pos int, rlpData []byte) (dataLen, newPos int, err error) {
	longPrefix := longString
	if isList {
		longPrefix = longList
	}
	lenOfLen := int(prefixByte - longPrefix) // assured to be <8
	pos++
	if lenOfLen > len(rlpData)-pos {
		return -1, -1, fmt.Errorf("length mismatch in RLP for length bytes (list=%t pos=%d len=%d)", isList, pos, lenOfLen)
	}
	dataLen, err = minimalBytesToInt64(rlpData[pos : pos+lenOfLen])
	if err != nil {
		return -1, -1, err
	}
	pos += lenOfLen
	if dataLen > len(rlpData)-pos {
		return -1, -1, fmt.Errorf("length mismatch in RLP for data bytes (list=%t pos=%d len=%d)", isList, pos, dataLen)
	}
	return dataLen, pos, nil
}

func minimalBytesToInt64(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	pow := len(data) - 1
	var v int64
	for i := 0; i < len(data); i++ {
		v += int64(data[i]) << (8 * pow)
		pow--
	}
	if v < 0 || v > maxInt32 {
		// We refuse to decode more than 2^32-1 of data
		return -1, fmt.Errorf("too many RLP bytes to decode")
	}
	return int(v), nil
}
