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

func encodeBytes(inBytes []byte, isList bool) []byte {
	shortOffset := shortString
	if isList {
		shortOffset = shortList
	}
	if len(inBytes) == 1 &&
		!isList &&
		inBytes[0] <= 0x7f {
		// We don't need the offset, this can be sent as a single byte
		return inBytes
	}
	if len(inBytes) <= 55 {
		// Add the length to same byte as the offset
		outBytes := make([]byte, len(inBytes)+1)
		outBytes[0] = shortOffset + byte(len(inBytes))
		copy(outBytes[1:], inBytes[0:])
		return outBytes
	}
	// The length is too long to fit in a single byte, we have to encode it
	encodedByteLen := int64ToMinimalBytes(int64(len(inBytes)))
	outBytes := make([]byte, 1+len(encodedByteLen)+len(inBytes))
	outBytes[0] = shortOffset + shortToLong + byte(len(encodedByteLen))
	copy(outBytes[1:], encodedByteLen)
	copy(outBytes[1+len(encodedByteLen):], inBytes)
	return outBytes
}

func int64ToMinimalBytes(v int64) []byte {
	vb := int64ToBytes(v)
	for i := 0; i < len(vb); i++ {
		if vb[i] != 0x00 {
			return vb[i:]
		}
	}
	return []byte{}
}

func int64ToBytes(v int64) [8]byte {
	return [8]byte{
		(byte)((v >> 56) & 0xff),
		(byte)((v >> 48) & 0xff),
		(byte)((v >> 40) & 0xff),
		(byte)((v >> 32) & 0xff),
		(byte)((v >> 24) & 0xff),
		(byte)((v >> 16) & 0xff),
		(byte)((v >> 8) & 0xff),
		(byte)(v & 0xff),
	}
}
