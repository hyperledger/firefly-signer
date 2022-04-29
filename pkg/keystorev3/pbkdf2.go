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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"golang.org/x/crypto/pbkdf2"
)

const (
	prfHmacSHA256 = "hmac-sha256"
)

func readPbkdf2WalletFile(jsonWallet []byte, password []byte) (WalletFile, error) {
	var w *walletFilePbkdf2
	if err := json.Unmarshal(jsonWallet, &w); err != nil {
		return nil, fmt.Errorf("invalid pbkdf2 keystore: %s", err)
	}
	return w, w.decrypt(password)
}

func (w *walletFilePbkdf2) decrypt(password []byte) (err error) {
	if w.Crypto.KDFParams.PRF != prfHmacSHA256 {
		return fmt.Errorf("invalid pbkdf2 wallet file: unsupported prf '%s'", w.Crypto.KDFParams.PRF)
	}

	derivedKey := pbkdf2.Key(password, w.Crypto.KDFParams.Salt, w.Crypto.KDFParams.C, w.Crypto.KDFParams.DKLen, sha256.New)

	privateKey, err := w.Crypto.decryptCommon(derivedKey)
	if err == nil {
		w.keypair, err = secp256k1.NewSecp256k1KeyPair(privateKey)
	}
	return err

}
