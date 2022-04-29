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
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

const privateKeySize = 32

type KeyPair struct {
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
	Address    ethtypes.Address
}

func (k *KeyPair) PrivateKeyBytes() []byte {
	return k.PrivateKey.D.FillBytes(make([]byte, privateKeySize))
}

func (k *KeyPair) PublicKeyBytes() []byte {
	// Remove the "04" Suffix byte when computing the address. This byte indicates that it is an uncompressed public key.
	return k.PublicKey.SerializeUncompressed()[1:]
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
	return &KeyPair{
		PrivateKey: key,
		PublicKey:  pubKey,
		Address:    *PublicKeyToAddress(pubKey),
	}
}

func PublicKeyToAddress(pubKey *btcec.PublicKey) *ethtypes.Address {
	// Take the hash of the public key to generate the address
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKey.SerializeUncompressed()[1:])
	// Ethereum addresses only use the lower 20 bytes, so toss the rest away
	a := new(ethtypes.Address)
	copy(a[:], hash.Sum(nil)[12:32])
	return a
}
