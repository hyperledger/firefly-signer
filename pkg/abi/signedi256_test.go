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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt256TwosCompliment(t *testing.T) {

	i := big.NewInt(-12345)
	b := SerializeInt256TwosComplementBytes(i)
	i2 := ParseInt256TwosComplementBytes(b)
	assert.Equal(t, int64(-12345), i2.Int64())

	// Largest negative two's compliment - 2^255
	i = new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil)
	i = i.Neg(i)
	b = SerializeInt256TwosComplementBytes(i)
	i3 := ParseInt256TwosComplementBytes(b)
	assert.Zero(t, i.Cmp(i3))

	// Largest positive two's compliment - 2^255-1
	i = new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil)
	i = i.Sub(i, big.NewInt(1))
	b = SerializeInt256TwosComplementBytes(i)
	i4 := ParseInt256TwosComplementBytes(b)
	assert.Zero(t, i.Cmp(i4))

}

func TestBitLen(t *testing.T) {

	assert.True(t, checkSignedIntFits(big.NewInt(0), 0))
	assert.False(t, checkSignedIntFits(big.NewInt(1), 0))

	assert.True(t, checkSignedIntFits(big.NewInt(-32768), 16))
	assert.False(t, checkSignedIntFits(big.NewInt(-32769), 16))

	assert.True(t, checkSignedIntFits(big.NewInt(32767), 16))
	assert.False(t, checkSignedIntFits(big.NewInt(32768), 16))

}
