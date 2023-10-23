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

// func TestSimpleStruct(t *testing.T) {

// 	// struct Mail {
// 	// 	address from;
// 	// 	address to;
// 	// 	string contents;
// 	// }
// 	mailABI := []byte(`{
// 		"components": [
// 			{
// 				"internalType": "address",
// 				"name": "from",
// 				"type": "address"
// 			},
// 			{
// 				"internalType": "address",
// 				"name": "to",
// 				"type": "address"
// 			},
// 			{
// 				"internalType": "string",
// 				"name": "contents",
// 				"type": "string"
// 			}
// 		],
// 		"internalType": "struct EIP712Examples.Mail",
// 		"name": "",
// 		"type": "tuple"
// 	}`)
// 	var abiElem abi.Parameter
// 	err := json.Unmarshal(mailABI, &abiElem)
// 	assert.NoError(t, err)

// 	tc, err := abiElem.TypeComponentTree()
// 	assert.NoError(t, err)

// 	pt, ts, err := ABItoEIP712TypeSet(context.Background(), tc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, "Mail", pt)
// 	assert.Equal(t, TypeSet{
// 		"Mail": Type{
// 			{
// 				Name: "from",
// 				Type: "address",
// 			},
// 			{
// 				Name: "to",
// 				Type: "address",
// 			},
// 			{
// 				Name: "contents",
// 				Type: "string",
// 			},
// 		},
// 	}, ts)

// 	cv, err := tc.ParseExternal([]interface{}{
// 		"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
// 		"0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
// 		"Hello, Bob!",
// 	})
// 	assert.NoError(t, err)

// 	eip712Type, err := EncodeTypeABI(context.Background(), cv.Component)
// 	assert.NoError(t, err)

// 	assert.Equal(t, "Mail(address from,address to,string contents)", eip712Type)

// 	encodedData, err := EncodeDataABI(context.Background(), cv)
// 	assert.NoError(t, err)

// 	assert.Equal(t, `0x`+
// 		`000000000000000000000000cd2a3d9f938e13cd947ec05abc7fe734df8dd826`+ // uint160 address
// 		`000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`+ // uint160 address
// 		`48656c6c6f2c20426f6221`, // ASCII: "Hello, Bob!"
// 		encodedData.String())

// 	hash, err := HashStructABI(context.Background(), cv)
// 	assert.NoError(t, err)

// 	assert.Equal(t, hashString(string(hashString(eip712Type))+string(encodedData)).String(), hash.String())

// }
