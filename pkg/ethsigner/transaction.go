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

package ethsigner

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

type TransactionSignaturePayload struct {
	rlpList rlp.List
	data    []byte
}

func (sp *TransactionSignaturePayload) Bytes() []byte {
	return sp.data
}

func (sp *TransactionSignaturePayload) Hash() ethtypes.HexBytes0xPrefix {
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(sp.data)
	return msgHash.Sum(nil)
}

const (
	TransactionTypeLegacy byte = 0x00
	TransactionType2930   byte = 0x01 // unused
	TransactionType1559   byte = 0x02
)

type Transaction struct {
	From                 json.RawMessage           `json:"from,omitempty"` // only here as a possible input to signing key selection (eth_sendTransaction)
	Nonce                *ethtypes.HexInteger      `json:"nonce,omitempty"`
	GasPrice             *ethtypes.HexInteger      `json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas *ethtypes.HexInteger      `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *ethtypes.HexInteger      `json:"maxFeePerGas,omitempty"`
	GasLimit             *ethtypes.HexInteger      `json:"gas"`
	To                   *ethtypes.Address0xHex    `json:"to,omitempty"`
	Value                *ethtypes.HexInteger      `json:"value"`
	Data                 ethtypes.HexBytes0xPrefix `json:"data"`
}

func (t *Transaction) BuildLegacy() rlp.List {
	rlpList := make(rlp.List, 0, 6)
	rlpList = append(rlpList, rlp.WrapInt(t.Nonce.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasPrice.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasLimit.BigInt()))
	rlpList = append(rlpList, rlp.WrapAddress(t.To))
	rlpList = append(rlpList, rlp.WrapInt(t.Value.BigInt()))
	rlpList = append(rlpList, rlp.Data(t.Data))
	return rlpList
}

func (t *Transaction) AddEIP155HashValues(rlpList rlp.List, chainID int64) rlp.List {
	// These values go into the hash of the transaction
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(chainID)))
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(0)))
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(0)))
	return rlpList
}

func (t *Transaction) Build1559(chainID int64) rlp.List {
	rlpList := make(rlp.List, 0, 9)
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(chainID)))
	rlpList = append(rlpList, rlp.WrapInt(t.Nonce.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.MaxPriorityFeePerGas.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.MaxFeePerGas.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasLimit.BigInt()))
	rlpList = append(rlpList, rlp.WrapAddress(t.To))
	rlpList = append(rlpList, rlp.WrapInt(t.Value.BigInt()))
	rlpList = append(rlpList, rlp.Data(t.Data))
	rlpList = append(rlpList, rlp.List{}) // access list not currently supported
	return rlpList
}

// Automatically pick signer, based on input fields.
// - If either of the new EIP-1559 fields are set, use EIP-1559
// - By default use EIP-155 signing
// Never picks legacy-legacy (non EIP-155), or EIP-2930
func (t *Transaction) Sign(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}
	if t.MaxPriorityFeePerGas.BigInt().Sign() > 0 || t.MaxFeePerGas.BigInt().Sign() > 0 {
		return t.SignEIP1559(signer, chainID)
	}
	return t.SignLegacyEIP155(signer, chainID)
}

// Returns the bytes that would be used to sign the transaction, without actually
// perform the signing. Can be used with Recover to verify a signing result.
func (t *Transaction) SignaturePayload(chainID int64) (sp *TransactionSignaturePayload) {
	if t.MaxPriorityFeePerGas.BigInt().Sign() > 0 || t.MaxFeePerGas.BigInt().Sign() > 0 {
		return t.SignaturePayloadEIP1559(chainID)
	}
	return t.SignaturePayloadLegacyEIP155(chainID)
}

// SignaturePayloadLegacyOriginal returns the rlpList of fields that are signed, and the
// bytes. Note that for legacy and EIP-155 transactions (everything prior to EIP-2718),
// there is no transaction type byte added (so the bytes are exactly rlpList.Encode())
func (t *Transaction) SignaturePayloadLegacyOriginal() *TransactionSignaturePayload {
	rlpList := t.BuildLegacy()
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    rlpList.Encode(),
	}
}

// SignLegacyOriginal uses legacy transaction structure, with legacy V value (27/28)
func (t *Transaction) SignLegacyOriginal(signer secp256k1.Signer) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}
	signatureData := t.SignaturePayloadLegacyOriginal()
	sig, err := signer.Sign(signatureData.data)
	if err != nil {
		return nil, err
	}

	rlpList := t.addSignature(signatureData.rlpList, sig)
	return rlpList.Encode(), nil
}

// SignaturePayloadLegacyEIP155 returns the rlpList of fields that are signed, and the
// bytes. Note that for legacy and EIP-155 transactions (everything prior to EIP-2718),
// there is no transaction type byte added (so the bytes are exactly rlpList.Encode())
func (t *Transaction) SignaturePayloadLegacyEIP155(chainID int64) *TransactionSignaturePayload {
	rlpList := t.BuildLegacy()
	rlpList = t.AddEIP155HashValues(rlpList, chainID)
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    rlpList.Encode(),
	}
}

// SignLegacyEIP155 uses legacy transaction structure, with EIP-155 signing V value (2*ChainID + 35 + Y-parity)
func (t *Transaction) SignLegacyEIP155(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}

	signaturePayload := t.SignaturePayloadLegacyEIP155(chainID)

	sig, err := signer.Sign(signaturePayload.data)
	if err != nil {
		return nil, err
	}

	// Use the EIP-155 V value, of (2*ChainID + 35 + Y-parity)
	sig.UpdateEIP155(chainID)

	rlpList := t.addSignature(signaturePayload.rlpList[0:6] /* we don't include the chainID+0+0 hash values in the payload */, sig)
	return rlpList.Encode(), nil
}

// SignaturePayloadEIP1559 returns the rlpList of fields that are signed, along with the full
// bytes for the signature / TX Hash - which have the transaction type prefixed
func (t *Transaction) SignaturePayloadEIP1559(chainID int64) *TransactionSignaturePayload {
	rlpList := t.Build1559(chainID)

	// The signature payload is the transaction type, concatenated with RLP list _excluding_ signature
	// keccak256(0x02 || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, destination, amount, data, access_list]))
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    append([]byte{TransactionType1559}, rlpList.Encode()...),
	}
}

// SignEIP1559 uses EIP-1559 transaction structure (with EIP-2718 transaction type byte), with EIP-2930 V value (0 / 1 - direct parity-Y)
func (t *Transaction) SignEIP1559(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}

	signaturePayload := t.SignaturePayloadEIP1559(chainID)
	sig, err := signer.Sign(signaturePayload.data)
	if err != nil {
		return nil, err
	}

	// Use the direct 0/1 Y-parity value
	sig.UpdateEIP2930()

	// Now we need a new RLP array, _including_ signature
	// 0x02 || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, destination, amount, data, access_list, signature_y_parity, signature_r, signature_s])
	rlpList := t.addSignature(signaturePayload.rlpList, sig)
	return append([]byte{TransactionType1559}, rlpList.Encode()...), nil
}

func (t *Transaction) addSignature(rlpList rlp.List, sig *secp256k1.SignatureData) rlp.List {
	rlpList = append(rlpList, rlp.WrapInt(sig.V))
	rlpList = append(rlpList, rlp.WrapInt(sig.R))
	rlpList = append(rlpList, rlp.WrapInt(sig.S))
	return rlpList
}
