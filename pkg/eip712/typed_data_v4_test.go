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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const request1 = `{
	"types": {
		"EIP712Domain": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "version",
				"type": "string"
			},
			{
				"name": "chainId",
				"type": "uint256"
			},
			{
				"name": "verifyingContract",
				"type": "address"
			}
		],
		"Person": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "wallet",
				"type": "address"
			}
		],
		"Mail": [
			{
				"name": "from",
				"type": "Person"
			},
			{
				"name": "to",
				"type": "Person"
			},
			{
				"name": "contents",
				"type": "string"
			}
		]
	},
	"primaryType": "Mail",
	"domain": {
		"name": "Ether Mail",
		"version": "1",
		"chainId": 1,
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	},
	"message": {
		"from": {
			"name": "Cow",
			"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
		},
		"to": {
			"name": "Bob",
			"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
		},
		"contents": "Hello, Bob!"
	}
}`

func TestMessage1(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p SignTypedDataPayload
	err := json.Unmarshal([]byte(request1), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.Equal(t, "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e", ed.String())

}
