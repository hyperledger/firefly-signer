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

package ed25519

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratedKeyRoundTrip(t *testing.T) {

	keypair, err := GenerateEd25519KeyPairLegacy()
	assert.NoError(t, err)

	b := keypair.PrivateKeyBytes()
	keypair2 := KeyPairFromBytesLegacy(b)
	assert.NotNil(t, keypair2)

	// For ed25519, we can reconstruct the keypair from private key bytes
	assert.Equal(t, keypair.PrivateKeyBytes(), keypair2.PrivateKeyBytes())
	assert.Equal(t, keypair.PublicKey, keypair2.PublicKey)

	data := []byte("hello world")
	sig, err := keypair.Sign(data)
	assert.NoError(t, err)

	// Test signature verification
	assert.True(t, sig.Verify(data, keypair.PublicKey))

	// Test compact signature encoding/decoding
	compactSig := sig.CompactSignature()
	sig2, err := DecodeCompactRS(context.Background(), compactSig)
	assert.NoError(t, err)
	assert.True(t, sig2.Verify(data, keypair.PublicKey))

	// Test error cases
	_, err = DecodeCompactRS(context.Background(), []byte("wrong"))
	assert.Error(t, err)

	// Test nil keypair
	_, err = (*KeyPair)(nil).Sign(data)
	assert.Error(t, err)

	// Test invalid private key length
	invalidKey := KeyPairFromBytesLegacy([]byte("short"))
	assert.Nil(t, invalidKey)
}

func TestEd25519KeyProperties(t *testing.T) {
	keypair, err := GenerateEd25519KeyPairLegacy()
	assert.NoError(t, err)

	// Check key sizes
	assert.Equal(t, 64, len(keypair.PrivateKey)) // ed25519 private key is 64 bytes
	assert.Equal(t, 32, len(keypair.PublicKey))  // ed25519 public key is 32 bytes
}

func TestSignatureVerification(t *testing.T) {
	keypair, err := GenerateEd25519KeyPairLegacy()
	assert.NoError(t, err)

	message := []byte("test message")
	sig, err := keypair.Sign(message)
	assert.NoError(t, err)

	// Verify correct signature
	assert.True(t, sig.Verify(message, keypair.PublicKey))

	// Verify wrong message
	assert.False(t, sig.Verify([]byte("wrong message"), keypair.PublicKey))

	// Verify with wrong R and S values
	wrongSig := &SignatureData{
		R: new(big.Int).SetInt64(0),
		S: new(big.Int).SetInt64(0),
	}
	assert.False(t, wrongSig.Verify(message, keypair.PublicKey))
}

func TestCompactSignatureFormat(t *testing.T) {
	keypair, err := GenerateEd25519KeyPairLegacy()
	assert.NoError(t, err)

	message := []byte("compact test")
	sig, err := keypair.Sign(message)
	assert.NoError(t, err)

	compactSig := sig.CompactSignature()
	assert.Equal(t, 64, len(compactSig)) // R (32 bytes) + S (32 bytes)

	// Test decoding
	sig2, err := DecodeCompactRS(context.Background(), compactSig)
	assert.NoError(t, err)
	assert.Equal(t, sig.R, sig2.R)
	assert.Equal(t, sig.S, sig2.S)

	// Test verification of decoded signature
	assert.True(t, sig2.Verify(message, keypair.PublicKey))
}
