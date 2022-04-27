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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hyperledger/firefly-signer/internal/secp256k1"
	"github.com/hyperledger/firefly-signer/internal/types"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/sha3"
)

const (
	nLight        int = 1 << 12
	pLight        int = 0
	nStandard     int = 1 << 10
	pStandard     int = 1
	r             int = 8
	saltLen       int = 32
	derivedKeyLen int = 32
	encryptKeyLen int = 16
	ivLen         int = 16 // 128bit
	version3      int = 3

	cipherAES128ctr = "es-128-ctr"
	kdfTypeScrypt   = "scrypt"
)

func newWalletFile(password string, keypair *secp256k1.KeyPair, n int, p int) WalletFile {

	salt := mustReadBytes(saltLen, rand.Reader)

	derivedKey := mustGenerateDerivedScryptKey(password, salt, n, p)

	encryptKey := derivedKey[0:encryptKeyLen]

	iv := mustReadBytes(ivLen, rand.Reader)

	cipherText := mustAES128CtrEncrypt(keypair.PrivateKeyBytes(), iv, encryptKey)

	mac := generateMac(derivedKey, cipherText)

	return &walletFileScrypt{
		walletFileBase: walletFileBase{
			Address: types.EthAddressPlainHex(keypair.Address),
			ID:      fftypes.NewUUID(),
			Version: version3,
		},
		Crypto: cryptoScrypt{
			cryptoCommon: cryptoCommon{
				Cipher:     cipherAES128ctr,
				CipherText: cipherText,
				CipherParams: cipherParams{
					IV: iv,
				},
				KDF: kdfTypeScrypt,
				MAC: mac,
			},
			KDFParams: scryptKdfParams{
				DKLen: len(derivedKey),
				N:     n,
				P:     p,
				R:     r,
				Salt:  salt,
			},
		},
	}
}

func NewWalletFileLight(password string, keypair *secp256k1.KeyPair) WalletFile {
	return newWalletFile(password, keypair, nLight, pLight)
}

func NewWalletFileStandard(password string, keypair *secp256k1.KeyPair) WalletFile {
	return newWalletFile(password, keypair, nStandard, pStandard)
}

func ReadWalletFile(jsonWallet []byte) (WalletFile, error) {
	var w walletFileCommon
	if err := json.Unmarshal(jsonWallet, &w); err != nil {
		return nil, fmt.Errorf("invalid wallet file: %s", err)
	}
	if w.ID == nil {
		return nil, fmt.Errorf("missing keyfile ID")
	}
	if w.Version != version3 {
		return nil, fmt.Errorf("incorrect keyfile version (only V3 supported): %d", w.Version)
	}
	switch w.Crypto.KDF {
	case kdfTypeScrypt:
		return readScryptWalletFile(jsonWallet)
	default:
		return nil, fmt.Errorf("unsupported kdf: %s", w.Crypto.KDF)
	}
}

func readScryptWalletFile(jsonWallet []byte) (WalletFile, error) {
	var w walletFileScrypt
	if err := json.Unmarshal(jsonWallet, &w); err != nil {
		return nil, fmt.Errorf("invalid scrypt wallet file: %s", err)
	}
	return &w, nil
}

func mustReadBytes(size int, r io.Reader) []byte {
	b := make([]byte, size)
	n, err := io.ReadFull(r, b)
	if err != nil || n != size {
		panic(fmt.Sprintf("Read failed (len=%d): %s", n, err))
	}
	return b
}

func mustGenerateDerivedScryptKey(password string, salt []byte, n, p int) []byte {
	b, err := scrypt.Key([]byte(password), salt, n, r, p, derivedKeyLen)
	if err != nil {
		panic(fmt.Sprintf("Scrypt failed: %s", err))
	}
	return b
}

func mustAES128CtrEncrypt(key []byte, iv []byte, plaintext []byte) []byte {

	// Per https://go.dev/src/crypto/cipher/example_test.go ExampleNewCTR

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("AES initialization failed: %s", err))
	}

	ciphertext := make([]byte, len(iv)+len(key))

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[len(iv):], plaintext)

	return ciphertext

}

func generateMac(derivedKey []byte, cipherText []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(derivedKey[0:16])
	hash.Write(cipherText)
	return hash.Sum(nil)
}
