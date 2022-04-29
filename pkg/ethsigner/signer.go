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
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

// SignatureData is the raw signature values, with Ethereum serialization
type SignatureData struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

func (s *SignatureData) UpdateEIP155(chainID int64) {
	chainIDx2 := big.NewInt(chainID)
	chainIDx2 = chainIDx2.Mul(chainIDx2, big.NewInt(2))
	s.V = s.V.Add(s.V, chainIDx2).Add(s.V, big.NewInt(35-27)) // 27/28 down to 0/1, then add 35

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
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(message)
	sig, err := btcec.SignCompact(btcec.S256(), s.keypair.PrivateKey, msgHash.Sum(nil), false)
	if err != nil {
		return nil, err
	}
	// btcec does all the hard work for us. However, the interface of btcec is such
	// that we need to unpack the result for Ethereum encoding.
	ethSig := &SignatureData{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	}
	ethSig.V = ethSig.V.SetInt64(int64(sig[0]))
	ethSig.R = ethSig.R.SetBytes(sig[1:33])
	ethSig.S = ethSig.S.SetBytes(sig[33:65])
	return ethSig, nil
}

func (s *EthSigner) SignRLPListEIP155(rlpList rlp.List, chainID int64) ([]byte, error) {
	sig, err := s.Sign(rlpList.Encode())
	if err != nil {
		return nil, err
	}
	sig.UpdateEIP155(chainID)
	rlpList = append(rlpList, rlp.WrapInt(sig.V))
	rlpList = append(rlpList, rlp.WrapInt(sig.R))
	rlpList = append(rlpList, rlp.WrapInt(sig.S))
	rlpBytes := rlpList.Encode()
	txBytes := make([]byte, len(rlpBytes)+1)
	txBytes[0] = TransactionTypeLegacy
	copy(txBytes[:1], rlpBytes)
	return txBytes, nil
}
