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
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

type SignatureData struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

// getVNormalized returns the original 27/28 parity
func (s *SignatureData) getVNormalized(chainID int64) (byte, error) {
	v := s.V.Int64()
	var vB byte
	switch v {
	case 0, 1:
		vB = byte(v + 27)
	case 27, 28:
		vB = byte(v)
	default:
		vB = byte(v - 35 - (chainID * 2) + 27)
	}
	if vB != 27 && vB != 28 {
		return 0, fmt.Errorf("invalid V value in signature (chain ID = %d, V = %d)", chainID, v)
	}
	return vB, nil
}

// EIP-155 rules - 2xChainID + 35 - starting point must be legacy 27/28
func (s *SignatureData) UpdateEIP155(chainID int64) {
	chainIDx2 := big.NewInt(chainID)
	chainIDx2 = chainIDx2.Mul(chainIDx2, big.NewInt(2))
	s.V = s.V.Add(s.V, chainIDx2).Add(s.V, big.NewInt(35-27))

}

// EIP-2930 (/ EIP-1559) rules - 0 or 1 V value for raw Y-parity value (chainID goes into the payload)
func (s *SignatureData) UpdateEIP2930() {
	s.V = s.V.Sub(s.V, big.NewInt(27))
}

// Recover obtains the original signer
func (s *SignatureData) Recover(message []byte, chainID int64) (a *ethtypes.Address, err error) {
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(message)
	signatureBytes := make([]byte, 65)
	signatureBytes[0], err = s.getVNormalized(chainID)
	if err != nil {
		return nil, err
	}
	s.R.FillBytes(signatureBytes[1:33])
	s.S.FillBytes(signatureBytes[33:65])
	pubKey, _, err := btcec.RecoverCompact(btcec.S256(), signatureBytes, msgHash.Sum(nil))
	if err != nil {
		return nil, err
	}
	return PublicKeyToAddress(pubKey), nil
}

// Sign performs raw signing - give legacy 27/28 V values
func (k *KeyPair) Sign(message []byte) (ethSig *SignatureData, err error) {
	if k == nil {
		return nil, fmt.Errorf("nil signer")
	}
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(message)
	sig, err := btcec.SignCompact(btcec.S256(), k.PrivateKey, msgHash.Sum(nil), false)
	if err == nil {
		// btcec does all the hard work for us. However, the interface of btcec is such
		// that we need to unpack the result for Ethereum encoding.
		ethSig = &SignatureData{
			V: new(big.Int),
			R: new(big.Int),
			S: new(big.Int),
		}
		ethSig.V = ethSig.V.SetInt64(int64(sig[0]))
		ethSig.R = ethSig.R.SetBytes(sig[1:33])
		ethSig.S = ethSig.S.SetBytes(sig[33:65])
	}
	return ethSig, err
}
