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
	"encoding/json"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/stretchr/testify/assert"
)

func TestScryptWalletRoundTripLightForSecp256k1(t *testing.T) {
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	w1 := NewWalletFileLight("waltsentme", keypair.PrivateKeyBytes(), keypair.Address.String())
	assert.Equal(t, keypair.PrivateKeyBytes(), w1.PrivateKey())

	w1b, err := json.Marshal(&w1)
	assert.NoError(t, err)

	w2, err := ReadWalletFile(w1b, []byte("waltsentme"))
	assert.NoError(t, err)
	assert.Equal(t, keypair.PrivateKeyBytes(), w2.PrivateKey())

}

func TestScryptWalletRoundTripStandard(t *testing.T) {
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	w1 := NewWalletFileStandard("TrustNo1", keypair.PrivateKeyBytes(), keypair.Address.String())
	assert.Equal(t, keypair.PrivateKeyBytes(), w1.PrivateKey())

	w1b, err := json.Marshal(&w1)
	assert.NoError(t, err)

	w2, err := ReadWalletFile(w1b, []byte("TrustNo1"))
	assert.NoError(t, err)
	assert.Equal(t, keypair.PrivateKeyBytes(), w2.PrivateKey())

}

func TestScryptWalletRoundTripLightForBabyjubjub(t *testing.T) {
	privKey := babyjub.NewRandPrivKey()
	pubKey := privKey.Public()

	w1 := NewWalletFileLight("waltsentme", privKey[:], pubKey.Compress().String())
	assert.Equal(t, privKey[:], w1.PrivateKey())

	w1b, err := json.Marshal(&w1)
	assert.NoError(t, err)

	w2, err := ReadWalletFile(w1b, []byte("waltsentme"))
	assert.NoError(t, err)
	var recoveredPrivKey babyjub.PrivateKey
	copy(recoveredPrivKey[:], w2.PrivateKey())
	assert.Equal(t, recoveredPrivKey.Public().Compress(), pubKey.Compress())

}

func TestScryptReadInvalidFile(t *testing.T) {

	_, err := readScryptWalletFile([]byte(`!bad JSON`), []byte(""))
	assert.Error(t, err)

}

func TestMustGenerateDerivedScryptKeyPanic(t *testing.T) {

	assert.Panics(t, func() {
		mustGenerateDerivedScryptKey("", nil, 0, 1)
	})

}

func TestScryptWalletFileDecryptInvalid(t *testing.T) {

	w := &walletFileScrypt{}
	err := w.decrypt([]byte(""))
	assert.Regexp(t, "invalid scrypt keystore", err)

}

func TestScryptWalletFileDecryptInvalidDKLen(t *testing.T) {

	var w *walletFileScrypt
	err := json.Unmarshal([]byte(sampleWallet), &w)
	assert.NoError(t, err)

	w.Crypto.KDFParams.DKLen = 16
	err = w.decrypt([]byte("test"))
	assert.Regexp(t, "derived key length", err)

}

func TestScryptWalletFileDecryptBadPassword(t *testing.T) {

	var w *walletFileScrypt
	err := json.Unmarshal([]byte(sampleWallet), &w)
	assert.NoError(t, err)

	err = w.decrypt([]byte("wrong"))
	assert.Regexp(t, "invalid password", err)

}
