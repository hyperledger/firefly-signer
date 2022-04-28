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
	"encoding/hex"
	"math/big"
	"strings"
)

type Data []byte

type List []Element

type Element interface {
	IsList() bool
	Encode() []byte
}

func WrapString(s string) Data {
	return Data(s)
}

func WrapInt(i *big.Int) Data {
	return Data(i.Bytes())
}

func WrapHex(s string) (Data, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}
	return Data(b), nil
}

func MustWrapHex(s string) Data {
	b, err := WrapHex(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (r Data) Int() *big.Int {
	if r == nil {
		return nil
	}
	i := new(big.Int)
	return i.SetBytes(r)
}

func (r Data) Encode() []byte {
	return encodeBytes(r, false)
}

func (r Data) IsList() bool {
	return false
}

func (l List) Encode() []byte {
	if len(l) == 0 {
		return encodeBytes([]byte{}, true)
	}
	var concatenation []byte
	for _, entry := range l {
		concatenation = append(concatenation, entry.Encode()...)
	}
	return encodeBytes(concatenation, true)

}

func (l List) IsList() bool {
	return true
}
