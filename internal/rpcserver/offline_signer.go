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

package rpcserver

import (
	"context"
	"encoding/json"
	"os"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rpcbackend"
)

type SignTransactionInput struct {
	ethsigner.Transaction
	Address *ethtypes.Address0xHex `json:"address"`
	ABI     *abi.ABI               `json:"abi"`
	Method  string                 `json:"method,omitempty"`
	Params  interface{}            `json:"params"`
}

func (s *rpcServer) SignTransactionFromFile(ctx context.Context, filename string) error {

	// Parse the input
	var input SignTransactionInput
	inputData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(inputData, &input); err != nil {
		return err
	}

	// Find the function to invoke
	functions := input.ABI.Functions()
	var method *abi.Entry
	if input.Method == "" {
		if len(functions) != 1 {
			return i18n.NewError(ctx, signermsgs.MsgOfflineSignMethodCount, len(functions))
		}
	} else {
		for _, f := range functions {
			if f.String() == input.Method {
				// Full signature match wins
				method = f
			} else if method == nil && f.Name == input.Method {
				// But name is good enough (could be multiple overrides, so we don't break)
				method = f
			}
		}
		if method == nil {
			return i18n.NewError(ctx, signermsgs.MsgOfflineSignMethodNotFound, input.Method)
		}
	}

	// Generate the transaction data
	paramValues, err := method.Inputs.ParseExternalDataCtx(ctx, input.Params)
	if err == nil {
		input.Data, err = method.EncodeCallDataCtx(ctx, paramValues)
	}
	if err != nil {
		return err
	}

	// Sign the transaction
	rpcRes, err := s.processEthTransaction(ctx, &rpcbackend.RPCRequest{}, &input.Transaction)
	if err != nil {
		return err
	}
	resBytes, _ := json.Marshal(rpcRes)
	log.L(ctx).Infof("Submitted:\n%s", resBytes)
	return nil

}
