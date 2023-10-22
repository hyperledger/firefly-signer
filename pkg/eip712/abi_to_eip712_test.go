// Copyright © 2023 Kaleido, Inc.
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

package eip712

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/stretchr/testify/assert"
)

func TestSimpleStruct(t *testing.T) {

	// struct Mail {
	// 	address from;
	// 	address to;
	// 	string contents;
	// }
	mailABI := []byte(`{
		"components": [
			{
				"internalType": "address",
				"name": "from",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "to",
				"type": "address"
			},
			{
				"internalType": "string",
				"name": "contents",
				"type": "string"
			}
		],
		"internalType": "struct EIP712Examples.Mail",
		"name": "",
		"type": "tuple"
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	cv, err := tc.ParseExternal([]interface{}{
		"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
		"0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
		"Hello, Bob!",
	})
	assert.NoError(t, err)

	eip712Type, err := EncodeTypeABI(context.Background(), cv.Component)
	assert.NoError(t, err)

	assert.Equal(t, "Mail(address from,address to,string contents)", eip712Type)

	encodedData, err := EncodeDataABI(context.Background(), cv)
	assert.NoError(t, err)

	assert.Equal(t, "0x12345", encodedData.String())

}
