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

package rpcbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly/pkg/ffresty"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
)

type RPCCode int64

const (
	RPCCodeParseError     RPCCode = -32700
	RPCCodeInvalidRequest RPCCode = -32600
	RPCCodeInternalError  RPCCode = -32603
)

// Backend performs communication with a backend
type Backend interface {
	CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) error
	SyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error)
}

// NewRPCBackend Constructor
func NewRPCBackend(ctx context.Context) Backend {
	return &rpcBackend{
		client: ffresty.New(ctx, signerconfig.BackendPrefix),
	}
}

type rpcBackend struct {
	client         *resty.Client
	requestCounter int64
}

type RPCRequest struct {
	JSONRpc string             `json:"jsonrpc"`
	ID      *fftypes.JSONAny   `json:"id"`
	Method  string             `json:"method"`
	Params  []*fftypes.JSONAny `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRpc string             `json:"jsonrpc"`
	ID      *fftypes.JSONAny   `json:"id"`
	Result  *fftypes.JSONAny   `json:"result,omitempty"`
	Code    int64              `json:"code"`
	Message string             `json:"message"`
	Data    []*fftypes.JSONAny `json:"data,omitempty"`
}

func (rb *rpcBackend) allocateRequestID(req *RPCRequest) {
	reqID := atomic.AddInt64(&rb.requestCounter, 1)
	req.ID = fftypes.JSONAnyPtr(fmt.Sprintf(`"%.9d"`, reqID))
}

func (rb *rpcBackend) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) error {
	req := &RPCRequest{
		JSONRpc: "2.0",
		Method:  method,
		Params:  make([]*fftypes.JSONAny, len(params)),
	}
	for i, param := range params {
		b, err := json.Marshal(param)
		if err != nil {
			return i18n.NewError(ctx, signermsgs.MsgInvalidParam, i, method, err)
		}
		req.Params[i] = fftypes.JSONAnyPtrBytes(b)
	}
	res, err := rb.SyncRequest(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(res.Result.Bytes(), &result)
}

// SyncRequest sends an individual RPC request to the backend (always over HTTP currently),
// and waits synchronously for the response, or an error.
//
// In all return paths *including error paths* the RPCResponse is populated
// so the caller has an RPC structure to send back to the front-end caller.
func (rb *rpcBackend) SyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error) {

	// We always set the back-end request ID - as we need to support requests coming in from
	// multiple concurrent clients on our front-end that might use clashing IDs.
	var beReq = *rpcReq
	beReq.JSONRpc = "2.0"
	rb.allocateRequestID(&beReq)
	rpcRes = new(RPCResponse)

	log.L(ctx).Infof("RPC:%s:%s --> %s", beReq.ID, rpcReq.ID, rpcReq.Method)
	res, err := rb.client.R().
		SetContext(ctx).
		SetBody(&beReq).
		SetResult(rpcRes).
		SetError(rpcRes).
		Post("/")
	// Restore the original ID
	rpcRes.ID = rpcReq.ID
	if err != nil {
		err := i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, err)
		log.L(ctx).Errorf("RPC:%s:%s <-- ERROR: %s", beReq.ID, rpcReq.ID, err)
		rpcRes = RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError)
		return rpcRes, err
	}
	if res.IsError() {
		log.L(ctx).Errorf("RPC:%s:%s <-- [%d]: %s", beReq.ID, rpcReq.ID, res.StatusCode(), rpcRes.Message)
		err := fmt.Errorf(rpcRes.Message)
		return rpcRes, err
	}
	log.L(ctx).Infof("RPC:%s:%s <-- [%d] OK", beReq.ID, rpcReq.ID, res.StatusCode())
	return rpcRes, nil
}

func RPCErrorResponse(err error, id *fftypes.JSONAny, code RPCCode) *RPCResponse {
	return &RPCResponse{
		JSONRpc: "2.0",
		ID:      id,
		Code:    int64(code),
		Message: err.Error(),
	}
}
