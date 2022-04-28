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
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

const (
	ethMessagePrefix = "\u0019Ethereum Signed Message:\n"
)

// SignatureData is the raw signature values, with Ethereum serialization
type SignatureData struct {
	V int64
	R []byte
	S []byte
}

func (s *SignatureData) EIP155SignatureData(chainID int64) (*SignatureData, error) {
	if s.V < 27 || s.V > 28 {
		return nil, fmt.Errorf("signature appears to already be EIP-155 encoded")
	}
	eip155V := (s.V - 27) + // 0 or 1
		(chainID * 2) +
		35
	return &SignatureData{
		V: eip155V,
		R: s.R,
		S: s.S,
	}, nil
}

type EthSigner struct {
	keypair *secp256k1.KeyPair
}

func NewSigner(keypair *secp256k1.KeyPair) *EthSigner {
	return &EthSigner{
		keypair: keypair,
	}
}

// Sign performs raw signing - see SignatureData functions for EIP-155 serialization
func (s *EthSigner) Sign(message []byte) (*SignatureData, error) {
	msgHash := s.ethMessageHash(message)
	sig, err := btcec.SignCompact(btcec.S256(), s.keypair.PrivateKey, msgHash, false)
	if err != nil {
		return nil, err
	}
	// btcec does all the hard work for us. However, the interface of btcec is such
	// that we need to unpack the result for Ethereum encoding.
	ethSig := &SignatureData{
		R: make([]byte, 32),
		S: make([]byte, 32),
	}
	ethSig.V = int64(sig[0])
	copy(ethSig.R, sig[1:33])
	copy(ethSig.S, sig[33:65])
	return ethSig, nil
}

func (s *EthSigner) ethMessageHash(message []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(ethMessagePrefix))
	hash.Write([]byte(strconv.FormatInt(int64(len(message)), 10)))
	hash.Write(message)
	return hash.Sum(nil)
}
