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
	"testing"

	"github.com/stretchr/testify/assert"
)

const samplePrivateKey = `f6d5b8eb66ac39a39004209b7da586e3f95ecd1265172850b15e305c5d1fe424`

const sampleWallet = `{
	"address": "5d093e9b41911be5f5c4cf91b108bac5d130fa83",
	"crypto": {
	  "cipher": "aes-128-ctr",
	  "ciphertext": "a28e5f6fd3189ef220f658392af0e967f17931530ac5b79376ed5be7d8adfb5a",
	  "cipherparams": {
		"iv": "7babf856e25f812d9dbc133e3122a1fc"
	  },
	  "kdf": "scrypt",
	  "kdfparams": {
		"dklen": 32,
		"n": 262144,
		"p": 1,
		"r": 8,
		"salt": "2844947e39e03785cad3ccda776279dbf5a86a5df9cb6d0ab5773bfcb7cbe3b7"
	  },
	  "mac": "69ed15cbb03a29ec194bdbd2c2d8084c62be620d5b3b0f668ed9aa1f45dbaf99"
	},
	"id": "307cc063-2344-426a-b992-3b72d5d5be0b",
	"version": 3
  }`

func TestLoadWallet(t *testing.T) {
	_, err := ReadWalletFile([]byte(sampleWallet))
	assert.NoError(t, err)
}
