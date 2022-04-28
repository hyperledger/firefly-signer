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

package secp256k1

import (
	"encoding/hex"
	"testing"

	"github.com/hyperledger/firefly-signer/internal/types"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/stretchr/testify/assert"
)

// Test data directly taken from:
// https://github.com/web3j/web3j/blob/master/crypto/src/test/java/org/web3j/crypto/SignTest.java
var (
	sampleMessage    = "A test message"
	samplePrivateKey = "a392604efc2fad9c0b3da43b5f698a2e3f270f170d859912be0d54742275c5f6"
	samplePublicKey  = "0x506bc1dc099358e5137292f4efdd57e400f29ba5132aa5d12b18dac1c1f6aab" +
		"a645c0b7b58158babbfa6c6cd5a48aa7340a8749176b120e8516216787a13dc76"
	sampleAddress = "0xef678007d18427e6022059dbc264f27507cd1ffc"
)

func testKeyPair(t *testing.T) *secp256k1.KeyPair {
	keyBytes, err := hex.DecodeString(samplePrivateKey)
	assert.NoError(t, err)
	keypair, err := secp256k1.NewSecp256k1KeyPair(keyBytes)
	assert.NoError(t, err)
	return keypair
}

func TestValidateSampleData(t *testing.T) {
	// Validate the above sample data is consistent in the base secp256k1 key management layer
	keypair := testKeyPair(t)
	assert.Equal(t, samplePrivateKey, ((types.HexBytesPlain)(keypair.PrivateKeyBytes())).String())
	var pubkey types.HexBytes0xPrefix = keypair.PublicKeyBytes()
	assert.Equal(t, samplePublicKey, pubkey.String())
	var addr types.EthAddress0xHex = types.EthAddress0xHex(keypair.Address)
	assert.Equal(t, sampleAddress, addr.String())
}

func TestSignMessage(t *testing.T) {

	signer := NewSigner(testKeyPair(t))
	sig, err := signer.Sign([]byte(sampleMessage))
	assert.NoError(t, err)

	assert.Equal(t, int64(28), sig.V)
	assert.Equal(t, "0464eee9e2fe1a10ffe48c78b80de1ed8dcf996f3f60955cb2e03cb21903d930", ((types.HexBytesPlain)(sig.R)).String())
	assert.Equal(t, "06624da478b3f862582e85b31c6a21c6cae2eee2bd50f55c93c4faad9d9c8d7f", ((types.HexBytesPlain)(sig.S)).String())

}
