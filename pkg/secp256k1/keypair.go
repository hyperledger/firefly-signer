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
	"github.com/btcsuite/btcd/btcec" // ISC licensed
	"github.com/hyperledger/firefly-signer/internal/types"
	"golang.org/x/crypto/sha3"
)

const privateKeySize = 32

type KeyPair struct {
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
	Address    types.EthAddress
}

func (k *KeyPair) PrivateKeyBytes() []byte {
	return k.PrivateKey.D.FillBytes(make([]byte, privateKeySize))
}

func GenerateSecp256k1KeyPair() (*KeyPair, error) {
	key, _ := btcec.NewPrivateKey(btcec.S256())
	return wrapSecp256k1Key(key, key.PubKey()), nil
}

func NewSecp256k1KeyPair(b []byte) (*KeyPair, error) {
	key, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), b)
	return wrapSecp256k1Key(key, pubKey), nil
}

func wrapSecp256k1Key(key *btcec.PrivateKey, pubKey *btcec.PublicKey) *KeyPair {
	k := &KeyPair{
		PrivateKey: key,
		PublicKey:  pubKey,
	}

	// Remove the "04" Suffix byte when computing the address. This byte indicates that it is an uncompressed public key.
	publicKeyBytes := k.PublicKey.SerializeUncompressed()[1:]
	// Take the hash of the public key to generate the address
	hash := sha3.NewLegacyKeccak256()
	hash.Write(publicKeyBytes)
	// Ethereum addresses only use the lower 20 bytes, so toss the rest away
	copy(k.Address[:], hash.Sum(nil)[12:32])

	return k
}
