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
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func mustAES128CtrEncrypt(key []byte, iv []byte, plaintext []byte) []byte {

	// Per https://go.dev/src/crypto/cipher/example_test.go ExampleNewCTR

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("AES initialization failed: %s", err))
	}

	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext

}

func aes128CtrDecrypt(key []byte, iv []byte, ciphertext []byte) ([]byte, error) {

	// Per https://go.dev/src/crypto/cipher/example_test.go ExampleNewCTR

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES initialization failed: %s", err)
	}

	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil

}
