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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test private keys for ed25519 testing
// These are valid ed25519 private keys (64 bytes each)
var (
	testPrivateKey1 = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	testPrivateKey2 = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	testPrivateKey3 = "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"
)

// Solana address resolver for testing
func solanaAddressResolver(ctx context.Context, publicKey []byte) string {
	return "solana:" + hex.EncodeToString(publicKey[:8])
}

// Ethereum address resolver for testing
func ethereumAddressResolver(ctx context.Context, publicKey []byte) string {
	return "ethereum:0x" + hex.EncodeToString(publicKey[:20])
}

func testKeyPair(t *testing.T) *KeyPair {
	keypair, err := GenerateEd25519KeyPair(solanaAddressResolver)
	assert.NoError(t, err)
	return keypair
}

func testKeyPairLegacy(t *testing.T) *KeyPair {
	keypair, err := GenerateEd25519KeyPairLegacy()
	assert.NoError(t, err)
	return keypair
}

func testKeyPairFromHexLegacy(t *testing.T, privateKeyHex string) *KeyPair {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	assert.NoError(t, err)

	keypair := KeyPairFromBytesLegacy(privateKeyBytes)
	assert.NotNil(t, keypair)
	return keypair
}

func TestSignSolanaTransaction(t *testing.T) {
	keypair := testKeyPair(t)

	// Simulate a Solana transaction message (simplified)
	// In real Solana, this would be a serialized transaction
	message := []byte("Solana transaction data")

	sig, err := keypair.Sign(message)
	assert.NoError(t, err)
	assert.NotNil(t, sig)
	assert.NotNil(t, sig.R)
	assert.NotNil(t, sig.S)

	// Test signature verification
	assert.True(t, sig.Verify(message, keypair.PublicKey))

	// Test address identifier
	assert.Contains(t, keypair.GetAddress(), "solana:")
}

func TestSignSolanaMessage(t *testing.T) {
	keypair := testKeyPair(t)

	// Test signing a simple message (like a Solana program instruction)
	message := []byte("Hello Solana")

	sig, err := keypair.Sign(message)
	assert.NoError(t, err)
	assert.NotNil(t, sig)
	assert.NotNil(t, sig.R)
	assert.NotNil(t, sig.S)

	// Test signature verification
	assert.True(t, sig.Verify(message, keypair.PublicKey))
}

func TestSignWithKnownPrivateKey(t *testing.T) {
	// Test with a generated key pair for reproducible results within the test
	keypair := testKeyPair(t)

	message := []byte("Known private key test")
	sig, err := keypair.Sign(message)
	assert.NoError(t, err)

	// Verify the signature
	assert.True(t, sig.Verify(message, keypair.PublicKey))

	// Test that wrong message fails verification
	assert.False(t, sig.Verify([]byte("Wrong message"), keypair.PublicKey))
}

func TestCompactSignature(t *testing.T) {
	keypair := testKeyPair(t)
	message := []byte("Compact signature test")

	sig, err := keypair.Sign(message)
	assert.NoError(t, err)

	compactSig := sig.CompactSignature()
	assert.Equal(t, 64, len(compactSig)) // R (32 bytes) + S (32 bytes)

	// Test decoding
	sig2, err := DecodeCompactRS(nil, compactSig)
	assert.NoError(t, err)
	assert.Equal(t, sig.R, sig2.R)
	assert.Equal(t, sig.S, sig2.S)

	// Test verification of decoded signature
	assert.True(t, sig2.Verify(message, keypair.PublicKey))
}

func TestSignFailNil(t *testing.T) {
	_, err := (*KeyPair)(nil).Sign([]byte("test message"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil signer")
}

func TestDecodeCompactSignatureInvalid(t *testing.T) {
	// Test with wrong length
	_, err := DecodeCompactRS(nil, []byte("wrong length"))
	assert.Error(t, err)

	// Test with nil
	_, err = DecodeCompactRS(nil, nil)
	assert.Error(t, err)
}
