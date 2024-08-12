// Copyright © 2024 Kaleido, Inc.
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
var posMax = map[uint16]*big.Int{}
var negMax = map[uint16]*big.Int{}

func init() {
	for i := 8; i <= 256; i += 8 {
		posMax[uint16(i)] = maxPositiveSignedInt(uint(i))
		negMax[uint16(i)] = maxNegativeSignedInt(uint(i))
	}
}

func maxPositiveSignedInt(bitLen uint) *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(singleBit, bitLen-1), big.NewInt(1))
}

func maxNegativeSignedInt(bitLen uint) *big.Int {
	return new(big.Int).Neg(new(big.Int).Lsh(singleBit, bitLen-1))
}

func checkSignedIntFits(i *big.Int, bitlen uint16) bool {
	switch i.Sign() {
	case 0:
		return true
	case 1:
		max, ok := posMax[bitlen]
		return ok && i.Cmp(max) <= 0
	default: // -1
		max, ok := negMax[bitlen]
		return ok && i.Cmp(max) >= 0
	}
}

func SerializeInt256TwosComplementBytes(i *big.Int) []byte {
	// Go doesn't have a function to serialize bytes in two's compliment,
	// but you can do a bitwise AND to get a positive integer containing
	// the bits of the two's compliment value (for the number of bits you provide)
	tcI := new(big.Int).And(i, fullBits256)
	b := make([]byte, 32)
	return tcI.FillBytes(b)
}

func ParseInt256TwosComplementBytes(b []byte) *big.Int {
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
