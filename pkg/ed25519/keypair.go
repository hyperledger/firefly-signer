// Copyright © 2025 Kaleido, Inc.
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
	"crypto/ed25519"
	cryptorand "crypto/rand"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/resolvers"
)

// KeyPair represents an ed25519 key pair with flexible address identification
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Address    string
}

func (k *KeyPair) PrivateKeyBytes() []byte {
	return k.PrivateKey
}

func (k *KeyPair) PublicKeyBytes() []byte {
	return k.PublicKey
}

func (k *KeyPair) GetAddress() string {
	return k.Address
}

// GenerateEd25519KeyPair generates a new ed25519 key pair with optional address resolution
func GenerateEd25519KeyPair(addressResolver resolvers.AddressResolver) (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		return nil, err
	}

	var addressID string
	if addressResolver != nil {
		addressID = addressResolver(context.Background(), publicKey)
	}

	return wrapEd25519Key(privateKey, publicKey, addressID), nil
}

// GenerateEd25519KeyPairLegacy generates a key pair without address resolution (backward compatibility)
func GenerateEd25519KeyPairLegacy() (*KeyPair, error) {
	return GenerateEd25519KeyPair(nil)
}

func KeyPairFromBytes(ctx context.Context, b []byte, addressResolver resolvers.AddressResolver) *KeyPair {
	if len(b) != ed25519.PrivateKeySize {
		return nil
	}
	privateKey := ed25519.PrivateKey(b)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	var address string
	if addressResolver != nil {
		address = addressResolver(ctx, publicKey)
	}

	return wrapEd25519Key(privateKey, publicKey, address)
}

// KeyPairFromBytesLegacy creates a key pair without address resolution (backward compatibility)
func KeyPairFromBytesLegacy(b []byte) *KeyPair {
	return KeyPairFromBytes(context.Background(), b, nil)
}

func wrapEd25519Key(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, addressID string) *KeyPair {
	// Create a zero address for backward compatibility
	var address ethtypes.Address0xHex

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address.String(),
	}
}
