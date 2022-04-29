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

package rpcserver

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-signer/internal/rpcbackend"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
)

func (s *rpcServer) processRPC(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {
	switch rpcReq.Method {
	case "eth_accounts", "personal_accounts":
		return s.processEthAccounts(ctx, rpcReq)
	case "eth_sendTransaction":
		return s.processEthSendTransaction(ctx, rpcReq)
	default:
		return s.backend.SyncRequest(ctx, rpcReq)
	}
}

func (s *rpcServer) processEthAccounts(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {
	accounts, err := s.wallet.GetAccounts(ctx)
	if err != nil {
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInternalError), err
	}
	b, _ := json.Marshal(&accounts)
	return &rpcbackend.RPCResponse{
		JSONRpc: "2.0",
		ID:      rpcReq.ID,
		Result:  fftypes.JSONAnyPtrBytes(b),
	}, nil
}

func (s *rpcServer) processEthSendTransaction(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {

	if len(rpcReq.Params) < 1 {
		err := i18n.NewError(ctx, signermsgs.MsgInvalidParamCount, 1, len(rpcReq.Params))
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInvalidRequest), err
	}

	var txn ethsigner.Transaction
	err := json.Unmarshal(rpcReq.Params[0].Bytes(), &txn)
	if err != nil {
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeParseError), err
	}

	if txn.From == nil {
		err := i18n.NewError(ctx, signermsgs.MsgMissingFrom)
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInvalidRequest), err
	}

	// We have trivial nonce management built-in for sequential signing API calls, by making a JSON/RPC request
	// to the up-stream node. This should not be relied upon for production use cases.
	// See FireFly Transaction Manager, or FireFly EthConnect, for more advanced nonce management capabilities.
	if txn.Nonce == nil {
		err = s.backend.CallRPC(ctx, &txn.Nonce, "eth_getTransactionCount", txn.From, "pending")
		if err != nil {
			return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInternalError), err
		}
	}

	return nil, nil
	// var hexData ethtypes.HexBytes0xPrefix
	// hexData, err = s.wallet.Sign(ctx, txn, config.)

}
