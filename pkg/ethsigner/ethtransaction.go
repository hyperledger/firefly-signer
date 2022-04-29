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

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
)

const (
	TransactionTypeLegacy byte = 0x00
	TransactionType2930   byte = 0x01 // unused
	TransactionType1559   byte = 0x02
)

type EthTransaction struct {
	Nonce                *ethtypes.HexInteger      `json:"nonce"`
	GasPrice             *ethtypes.HexInteger      `json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas *ethtypes.HexInteger      `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *ethtypes.HexInteger      `json:"maxFeePerGas,omitempty"`
	GasLimit             *ethtypes.HexInteger      `json:"gas"`
	To                   *ethtypes.Address         `json:"to,omitempty"`
	Value                *ethtypes.HexInteger      `json:"value"`
	Data                 ethtypes.HexBytes0xPrefix `json:"data"`
}

func (t *EthTransaction) SerializeLegacy() rlp.List {
	rlpList := make(rlp.List, 0, 6)
	rlpList = append(rlpList, rlp.WrapInt(t.Nonce.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasPrice.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasLimit.BigInt()))
	rlpList = append(rlpList, rlp.WrapAddress(t.To))
	rlpList = append(rlpList, rlp.WrapInt(t.Value.BigInt()))
	rlpList = append(rlpList, rlp.Data(t.Data))
	return rlpList
}

func (t *EthTransaction) Serialize1559(chainID int64) rlp.List {
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

func (t *EthTransaction) SignLegacyEIP155(signer *EthSigner, chainID int64) ([]byte, error) {
	return signer.SignRLPListEIP155(t.SerializeLegacy(), chainID)
}

func (t *EthTransaction) SignEIP1159(signer *EthSigner, chainID int64) ([]byte, error) {
	return signer.SignRLPListEIP155(t.Serialize1559(chainID), chainID)
}
