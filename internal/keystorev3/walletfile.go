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

package keystorev3

import (
	"github.com/hyperledger/firefly-signer/internal/secp256k1"
	"github.com/hyperledger/firefly-signer/internal/types"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type WalletFile interface {
	KeyPair() *secp256k1.KeyPair
}

type scryptKdfParams struct {
	DKLen int                 `json:"dklen"`
	N     int                 `json:"n"`
	P     int                 `json:"p"`
	R     int                 `json:"r"`
	Salt  types.HexBytesPlain `json:"salt"`
}

type aes128CtrKdfParams struct {
	DKLen int                 `json:"dklen"`
	C     int                 `json:"c"`
	PRF   string              `json:"prf"`
	Salt  types.HexBytesPlain `json:"salt"`
}

type cipherParams struct {
	IV types.HexBytesPlain `json:"iv"`
}

type cryptoCommon struct {
	Cipher       string              `json:"cipher"`
	CipherText   types.HexBytesPlain `json:"ciphertext"`
	CipherParams cipherParams        `json:"cipherparams"`
	KDF          string              `json:"kdf"`
	MAC          types.HexBytesPlain `json:"mac"`
}

type cryptoScrypt struct {
	cryptoCommon
	KDFParams scryptKdfParams `json:"kdfparams"`
}

type cryptoAES128Ctr struct {
	cryptoCommon
	KDFParams aes128CtrKdfParams `json:"kdfparams"`
}

type walletFileBase struct {
	Address types.EthAddressPlainHex `json:"address"`
	ID      *fftypes.UUID            `json:"id"`
	Version int                      `json:"version"`
}

type walletFileCommon struct {
	walletFileBase
	Crypto cryptoCommon `json:"crypto"`
}

type walletFileAES128Ctr struct {
	walletFileBase
	Crypto cryptoAES128Ctr `json:"crypto"`
}

type walletFileScrypt struct {
	walletFileBase
	Crypto cryptoScrypt `json:"crypto"`
}

func (w *walletFileBase) KeyPair() *secp256k1.KeyPair {
	return nil
}
