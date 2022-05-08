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
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBytesFixed(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes32"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	hexStr := "090807060504030201000908070605040302010009080706050403020100feed"
	b, err := hex.DecodeString(hexStr)
	data, dynamic, err := abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, hexStr, hex.EncodeToString(data))

}

func TestEncodeBytesDynamicExact32(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	lenHexStr := "0000000000000000000000000000000000000000000000000000000000000020"
	hexStr := "090807060504030201000908070605040302010009080706050403020100feed"
	b, err := hex.DecodeString(hexStr)
	data, dynamic, err := abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.NoError(t, err)
	assert.True(t, dynamic)
	assert.Equal(t, lenHexStr+hexStr, hex.EncodeToString(data))

}

func TestEncodeBytesDynamicLess32(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	lenHexStr := "0000000000000000000000000000000000000000000000000000000000000004"
	hexStr := "feedbeef"
	hexStrPadded := "feedbeef00000000000000000000000000000000000000000000000000000000"
	b, err := hex.DecodeString(hexStr)
	data, dynamic, err := abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.NoError(t, err)
	assert.True(t, dynamic)
	assert.Equal(t, lenHexStr+hexStrPadded, hex.EncodeToString(data))

}

func TestEncodeBytesDynamicMore32(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "string"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	lenHexStr := "0000000000000000000000000000000000000000000000000000000000000031"
	hexStr := "09080706050403020100090807060504030201000908070605040302010090807060504030201009080706050403020100"
	hexStrPadded := "09080706050403020100090807060504030201000908070605040302010090807060504030201009080706050403020100000000000000000000000000000000"
	b, err := hex.DecodeString(hexStr)
	data, dynamic, err := abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.NoError(t, err)
	assert.True(t, dynamic)
	assert.Equal(t, lenHexStr+hexStrPadded, hex.EncodeToString(data))

}

func TestEncodeStringWrongType(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "string"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	_, _, err = abiEncodeString(context.Background(), "test", bytes32Component, 12345)
	assert.Regexp(t, "FF22042", err)

}

func TestEncodeStringShort(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	lenHexStr := "000000000000000000000000000000000000000000000000000000000000000d"
	hexStr := "48656c6c6f2c20776f726c642100000000000000000000000000000000000000"
	data, dynamic, err := abiEncodeString(context.Background(), "test", bytes32Component, "Hello, world!")
	assert.NoError(t, err)
	assert.True(t, dynamic)
	assert.Equal(t, lenHexStr+hexStr, hex.EncodeToString(data))

}

func TestEncodeBytesFunction(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "function"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	hexStr := "72b88e6bd5be978cae4c29f83b0d7d360d255d42fce353f6"
	b, err := hex.DecodeString(hexStr)
	data, dynamic, err := abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, hexStr+"0000000000000000" /* padded 24 to 32 */, hex.EncodeToString(data))

}

func TestEncodeBytesFixedInsufficientInput(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes32"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	hexStr := "090807060504030201000908070605040302010009080706050403020100fe" // only 31 bytes
	b, err := hex.DecodeString(hexStr)
	_, _, err = abiEncodeBytes(context.Background(), "test", bytes32Component, b)
	assert.Regexp(t, "FF22043", err)

}

func TestEncodeBytesFixedWrongType(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "bytes32"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	_, _, err = abiEncodeBytes(context.Background(), "test", bytes32Component, 12345)
	assert.Regexp(t, "FF22042", err)

}

func TestEncodeSignedIntegerWrongType(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	_, _, err = abiEncodeSignedInteger(context.Background(), "test", bytes32Component, 12345)
	assert.Regexp(t, "FF22042", err)

}

func TestEncodeSignedIntegerPositiveOk(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	data, dynamic, err := abiEncodeSignedInteger(context.Background(), "test", bytes32Component, posMax[256])
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", hex.EncodeToString(data))

}

func TestEncodeSignedIntegerNegativeOk(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	data, dynamic, err := abiEncodeSignedInteger(context.Background(), "test", bytes32Component, negMax[256])
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, "8000000000000000000000000000000000000000000000000000000000000000", hex.EncodeToString(data))

}

func TestEncodeSignedIntegerTooLarge(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	i := new(big.Int).Add(posMax[256], big.NewInt(1))
	_, _, err = abiEncodeSignedInteger(context.Background(), "test", bytes32Component, i)
	assert.Regexp(t, "FF22044", err)

}

func TestEncodeSignedIntegerTooSmall(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	i := new(big.Int).Sub(negMax[256], big.NewInt(1))
	_, _, err = abiEncodeSignedInteger(context.Background(), "test", bytes32Component, i)
	assert.Regexp(t, "FF22044", err)

}

func TestEncodeUnsignedIntegerWrongType(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	_, _, err = abiEncodeUnsignedInteger(context.Background(), "test", bytes32Component, 12345)
	assert.Regexp(t, "FF22042", err)

}

func TestEncodeUnsignedIntegerPositiveOk(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	twoPow256minus1 := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	data, dynamic, err := abiEncodeUnsignedInteger(context.Background(), "test", bytes32Component, twoPow256minus1)
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", hex.EncodeToString(data))

}

func TestEncodeUnsignedIntegerTooLarge(t *testing.T) {

	bytes32Component, err := (&Parameter{Type: "int256"}).parseABIParameterComponents(context.Background())
	assert.NoError(t, err)

	twoPow256 := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	_, _, err = abiEncodeUnsignedInteger(context.Background(), "test", bytes32Component, twoPow256)
	assert.Regexp(t, "FF22044", err)

}
