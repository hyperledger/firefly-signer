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

package keystorev3

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
)

const (
	version3        = 3
	cipherAES128ctr = "aes-128-ctr"
	kdfTypeScrypt   = "scrypt"
	kdfTypePbkdf2   = "pbkdf2"
)

type WalletFile interface {
	PrivateKey() []byte
	KeyPair() *secp256k1.KeyPair
	JSON() []byte
}

type kdfParamsScrypt struct {
	DKLen int                    `json:"dklen"`
	N     int                    `json:"n"`
	P     int                    `json:"p"`
	R     int                    `json:"r"`
	Salt  ethtypes.HexBytesPlain `json:"salt"`
}

type kdfParamsPbkdf2 struct {
	DKLen int                    `json:"dklen"`
	C     int                    `json:"c"`
	PRF   string                 `json:"prf"`
	Salt  ethtypes.HexBytesPlain `json:"salt"`
}

type cipherParams struct {
	IV ethtypes.HexBytesPlain `json:"iv"`
}

type cryptoCommon struct {
	Cipher       string                 `json:"cipher"`
	CipherText   ethtypes.HexBytesPlain `json:"ciphertext"`
	CipherParams cipherParams           `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	MAC          ethtypes.HexBytesPlain `json:"mac"`
}

type cryptoScrypt struct {
	cryptoCommon
	KDFParams kdfParamsScrypt `json:"kdfparams"`
}

type cryptoPbkdf2 struct {
	cryptoCommon
	KDFParams kdfParamsPbkdf2 `json:"kdfparams"`
}

type walletFileBase struct {
	Address ethtypes.AddressPlainHex `json:"address"`
	ID      *fftypes.UUID            `json:"id"`
	Version int                      `json:"version"`

	privateKey []byte
	keypair    *secp256k1.KeyPair
}

type walletFileCommon struct {
	walletFileBase
	Crypto cryptoCommon `json:"crypto"`
}

type walletFilePbkdf2 struct {
	walletFileBase
	Crypto cryptoPbkdf2 `json:"crypto"`
}

type walletFileScrypt struct {
	walletFileBase
	Crypto cryptoScrypt `json:"crypto"`
}

func (w *walletFileBase) KeyPair() *secp256k1.KeyPair {
	return w.keypair
}

func (w *walletFileBase) PrivateKey() []byte {
	return w.privateKey
}

func (w *walletFilePbkdf2) JSON() []byte {
	b, _ := json.Marshal(w)
	return b
}

func (w *walletFileScrypt) JSON() []byte {
	b, _ := json.Marshal(w)
	return b
}

func (c *cryptoCommon) decryptCommon(derivedKey []byte) ([]byte, error) {
	if len(derivedKey) != 32 {
		return nil, fmt.Errorf("invalid scrypt keystore: derived key length %d != 32", len(derivedKey))
	}
	// Last 16 bytes of derived key are used for MAC
	derivedMac := generateMac(derivedKey[16:32], c.CipherText)
	if !bytes.Equal(derivedMac, c.MAC) {
		return nil, fmt.Errorf("invalid password provided")
	}
	// First 16 bytes of derived key are used as the encryption key
	encryptKey := derivedKey[0:16]
	return aes128CtrDecrypt(encryptKey, c.CipherParams.IV, c.CipherText)
}
