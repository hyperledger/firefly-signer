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
	"encoding/json"
	"fmt"
	"io"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

const (
	nLight    int = 1 << 12
	nStandard int = 1 << 10
	pDefault  int = 1
)

func NewWalletFileLight(password string, privateKey []byte, addr string) WalletFile {
	return newScryptWalletFile(password, privateKey, addr, nLight, pDefault)
}

func NewWalletFileStandard(password string, privateKey []byte, addr string) WalletFile {
	return newScryptWalletFile(password, privateKey, addr, nStandard, pDefault)
}

func addressFirst32(privateKey []byte) ethtypes.AddressPlainHex {
	if len(privateKey) > 32 {
		privateKey = privateKey[0:32]
	}
	kp, _ := secp256k1.NewSecp256k1KeyPair(privateKey)
	return ethtypes.AddressPlainHex(kp.Address)
}

func NewWalletFileCustomBytesLight(password string, privateKey []byte) WalletFile {
	return newScryptWalletFileBytes(password, privateKey, addressFirst32(privateKey).String(), nStandard, pDefault)
}

func NewWalletFileCustomBytesStandard(password string, privateKey []byte) WalletFile {
	return newScryptWalletFileBytes(password, privateKey, addressFirst32(privateKey).String(), nStandard, pDefault)
}

func ReadWalletFile(jsonWallet []byte, password []byte) (WalletFile, error) {
	var w walletFileCommon
	if err := json.Unmarshal(jsonWallet, &w); err != nil {
		return nil, fmt.Errorf("invalid wallet file: %s", err)
	}
	if w.ID == nil {
		return nil, fmt.Errorf("missing keyfile id")
	}
	if w.Version != version3 {
		return nil, fmt.Errorf("incorrect keyfile version (only V3 supported): %d", w.Version)
	}
	switch w.Crypto.KDF {
	case kdfTypeScrypt:
		return readScryptWalletFile(jsonWallet, password)
	case kdfTypePbkdf2:
		return readPbkdf2WalletFile(jsonWallet, password)
	default:
		return nil, fmt.Errorf("unsupported kdf: %s", w.Crypto.KDF)
	}
}

func mustReadBytes(size int, r io.Reader) []byte {
	b := make([]byte, size)
	n, err := io.ReadFull(r, b)
	if err != nil || n != size {
		panic(fmt.Sprintf("Read failed (len=%d): %s", n, err))
	}
	return b
}

func generateMac(derivedKeyMacBytes []byte, cipherText []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(derivedKeyMacBytes)
	hash.Write(cipherText)
	return hash.Sum(nil)
}
