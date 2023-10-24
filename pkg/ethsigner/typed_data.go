// Copyright © 2023 Kaleido, Inc.
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

package ethsigner

import (
	"context"

	"github.com/hyperledger/firefly-signer/pkg/eip712"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
)

func SignTypedDataV4(ctx context.Context, signer secp256k1.Signer, payload *eip712.TypedData) ([]byte, error) {
	encodedData, err := eip712.EncodeTypedDataV4(ctx, payload)
	if err != nil {
		return nil, err
	}
	sig, err := signer.Sign(encodedData)
	if err != nil {
		return nil, err
	}

	rlpList := make(rlp.List, 0, 4)
	rlpList = append(rlpList, rlp.Data(encodedData))
	rlpList = append(rlpList, rlp.WrapInt(sig.R))
	rlpList = append(rlpList, rlp.WrapInt(sig.S))
	rlpList = append(rlpList, rlp.WrapInt(sig.V))
	return rlpList.Encode(), nil
}
