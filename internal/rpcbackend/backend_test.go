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
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly/pkg/ffresty"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

type testRPCHander func(rpcReq *RPCRequest) (int, *RPCResponse)

func newTestServer(t *testing.T, rpcHandler testRPCHander) (context.Context, *rpcBackend, func()) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var rpcReq *RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReq)
		assert.NoError(t, err)

		status, rpcRes := rpcHandler(rpcReq)
		b := []byte(`{}`)
		if rpcRes != nil {
			b, err = json.Marshal(rpcRes)
			assert.NoError(t, err)
		}
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.WriteHeader(status)
		w.Write(b)

	}))

	signerconfig.Reset()
	prefix := signerconfig.BackendPrefix
	prefix.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", server.Listener.Addr()))

	rb := NewRPCBackend(ctx).(*rpcBackend)

	return ctx, rb, func() {
		cancelCtx()
		server.Close()
	}
}

func TestSyncRequestOK(t *testing.T) {

	rpcRequestBytes := []byte(`{
		"id": 2,
		"method": "eth_getTransactionByHash",
		"params": [
			"0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1"
		]
	}`)

	rpcServerResponseBytes := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"result": {
			"accessList": [],
			"blockHash": "0x471a236bac44222faf63e3d7808a2a68a704a75ca2f0774f072764867f458268",
			"blockNumber": "0xd536bc",
			"chainId": "0x1",
			"from": "0xfb075bb99f2aa4c49955bf703509a227d7a12248",
			"gas": "0x2b13d",
			"gasPrice": "0x3b6e7f5f09",
			"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
			"input": "0xa0712d680000000000000000000000000000000000000000000000000000000000000001",
			"maxFeePerGas": "0x4e58be5c3c",
			"maxPriorityFeePerGas": "0x59682f00",
			"nonce": "0x24",
			"r": "0xea6e1513d716146af3a02e1497fbe7fc3b2ffb08ccb4a1bfef4eaa2a122f62df",
			"s": "0xddc23aec20948a55d3e1f8afd29b5570d8d279450a472b55561ef6afe4a07ff",
			"to": "0x3c99f2a4b366d46bcf2277639a135a6d1288eceb",
			"transactionIndex": "0x1d",
			"type": "0x2",
			"v": "0x1",
			"value": "0x8e1bc9bf040000"
		}
	}`)

	var rpcRequest RPCRequest
	err := json.Unmarshal(rpcRequestBytes, &rpcRequest)
	assert.NoError(t, err)

	var rpcServerResponse RPCResponse
	err = json.Unmarshal(rpcServerResponseBytes, &rpcServerResponse)
	assert.NoError(t, err)

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionByHash", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1"`, rpcReq.Params[0].String())
		rpcServerResponse.ID = rpcReq.ID
		return 200, &rpcServerResponse
	})
	rb.requestCounter = 12345
	defer done()

	rpcRes, err := rb.SyncRequest(ctx, &rpcRequest)
	assert.NoError(t, err)
	assert.Equal(t, `2`, rpcRes.ID.String())
	assert.Equal(t, `0x24`, rpcRes.Result.JSONObject().GetString(`nonce`))
}

func TestSyncRPCCallOK(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionCount", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0xfb075bb99f2aa4c49955bf703509a227d7a12248"`, rpcReq.Params[0].String())
		assert.Equal(t, `"pending"`, rpcReq.Params[1].String())
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  fftypes.JSONAnyPtr(`"0x26"`),
		}
	})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.NoError(t, err)
	assert.Equal(t, int64(0x26), txCount.BigInt().Int64())
}

func TestSyncRPCCallErrorResponse(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 500, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Message: "pop",
		}
	})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "pop", err)
}

func TestSyncRPCCallErrorBadInput(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "test-bad-params", map[bool]bool{false: true})
	assert.Regexp(t, "FF20211", err)
}

func TestSyncRPCCallServerDown(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "net_version")
	assert.Regexp(t, "FF20212", err)
}
