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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratedKeyRoundTrip(t *testing.T) {

	keypair, err := GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	b := keypair.PrivateKeyBytes()
	keypair2, err := NewSecp256k1KeyPair(b)
	assert.NoError(t, err)

	assert.Equal(t, keypair.PrivateKeyBytes(), keypair2.PrivateKeyBytes())
	assert.True(t, keypair.PublicKey.IsEqual(keypair2.PublicKey))

}
