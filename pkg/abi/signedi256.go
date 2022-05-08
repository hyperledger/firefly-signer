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

import "math/big"

var singleBit = big.NewInt(1)
var oneMoreThanMaxUint256 = new(big.Int).Lsh(singleBit, 256)             // 2^256 - a one then 256 zeros
var fullBits256 = new(big.Int).Sub(oneMoreThanMaxUint256, big.NewInt(1)) // all ones for 256 bits
var oneThen255Zeros = new(big.Int).Lsh(singleBit, 255)

func serializeInt256TwosComplementBytes(i *big.Int) []byte {
	// Go doesn't have a function to serialize bytes in two's compliment,
	// but you can do a bitwise AND to get a positive integer containing
	// the bits of the two's compliment value (for the number of bits you provide)
	tcI := new(big.Int).And(i, fullBits256)
	b := make([]byte, 32)
	return tcI.FillBytes(b)
}

func parseInt256TwosComplementBytes(b []byte) *big.Int {
	// Parse the two's complement bytes as a positive number
	i := new(big.Int).SetBytes(b)
	// If the sign bit is not set, this is a positive number
	if i.Cmp(oneThen255Zeros) < 0 {
		return i
	}
	// Otherwise negate the value
	i.Sub(i, oneMoreThanMaxUint256)
	return i
}
