// Copyright © 2024 Kaleido, Inc.
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
	"fmt"
	"math/big"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

type SignatureData struct {
	R *big.Int
	S *big.Int
}

type Signer interface {
	// No hashing needed for ed25519 as other signers hash the message before signing
	Sign(message []byte) (*SignatureData, error)
}

// Verify verifies the signature against the message using the provided public key
func (s *SignatureData) Verify(message []byte, publicKey ed25519.PublicKey) bool {
	// Reconstruct the signature bytes from R and S
	signature := make([]byte, 64)
	copy(signature[:32], s.R.Bytes())
	copy(signature[32:], s.S.Bytes())

	// Use ed25519.Verify to check the signature
	return ed25519.Verify(publicKey, message, signature)
}

// CompactSignature returns the signature in a compact format
func (s *SignatureData) CompactSignature() []byte {
	// For ed25519, R and S are always exactly 32 bytes each
	result := make([]byte, 64)
	copy(result[:32], s.R.Bytes())
	copy(result[32:], s.S.Bytes())
	return result
}

func DecodeCompactRS(ctx context.Context, compactRS []byte) (*SignatureData, error) {
	if compactRS == nil {
		return nil, fmt.Errorf("compact signature cannot be nil")
	}

	if len(compactRS) != 64 {
		if ctx != nil {
			return nil, i18n.NewError(ctx, signermsgs.MsgSigningInvalidCompactRSV, len(compactRS))
		}
		return nil, fmt.Errorf("invalid compact signature length: expected 64, got %d", len(compactRS))
	}

	sig := &SignatureData{
		R: new(big.Int).SetBytes(compactRS[:32]),
		S: new(big.Int).SetBytes(compactRS[32:]),
	}
	return sig, nil
}

// SignDirect performs raw signing
func (k *KeyPair) Sign(message []byte) (sig *SignatureData, err error) {
	if k == nil {
		return nil, fmt.Errorf("nil signer")
	}

	signature := ed25519.Sign(k.PrivateKey, message)

	return &SignatureData{
		R: new(big.Int).SetBytes(signature[:32]),
		S: new(big.Int).SetBytes(signature[32:]),
	}, nil
}
