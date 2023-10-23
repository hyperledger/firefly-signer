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

const PersonType = `[{"name": "name","type": "string"},{"name": "wallet","type": "address"}]`

const MailType = `[{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]`

func TestMessage_ExampleFromEIP712Spec(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p SignTypedDataPayload
	err := json.Unmarshal([]byte(`{
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
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
		"domain": {
			"name": "Ether Mail",
			"version": "V4",
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
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xde26f53b35dd5ffdc13f8297e5cc7bbcb1a04bf33803bd2bf4a45eb251360cb8", ed.String())
}

func TestMessage_EmptyMessage(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p SignTypedDataPayload
	err := json.Unmarshal([]byte(`{
		"types": {},
		"primaryType": "EIP712Domain"
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x8d4a3f4082945b7879e2b55f181c31a77c8c0a464b70669458abbaaf99de4c38", ed.String())
}

func TestMessage_EmptyDomain(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p SignTypedDataPayload
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
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
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x25c3d40a39e639a4d0b6e4d2ace5e1281e039c88494d97d8d08f99a6ea75d775", ed.String())
}

func TestMessage_NilReference(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p SignTypedDataPayload
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": "Hello, Bob!"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x326faa52849c078e0e04abe863b29fc28d9d2885d2c4b515fcfb7ba1fac30534", ed.String())
}
