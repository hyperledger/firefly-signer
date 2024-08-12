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
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testRPCHandler func(rpcReq *RPCRequest) (int, *RPCResponse)
type testBatchRPCHandler func(rpcReq []*RPCRequest) (int, []*RPCResponse)

func newTestServer(t *testing.T, rpcHandler testRPCHandler, options ...RPCClientOptions) (context.Context, *RPCClient, func()) {

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
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", server.Listener.Addr()))

	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	var rb *RPCClient

	if len(options) == 1 {
		rb = NewRPCClientWithOption(c, options[0]).(*RPCClient)
	} else {
		rb = NewRPCClient(c).(*RPCClient)
	}

	return ctx, rb, func() {
		cancelCtx()
		server.Close()
	}
}

func newBatchTestServer(t *testing.T, rpcHandler testBatchRPCHandler, options ...RPCClientOptions) (context.Context, *RPCClient, func()) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var rpcReqs []*RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReqs)
		assert.NoError(t, err)

		status, rpcRes := rpcHandler(rpcReqs)
		b := []byte(`[]`)
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
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", server.Listener.Addr()))

	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	var rb *RPCClient

	if len(options) == 1 {
		rb = NewRPCClientWithOption(c, options[0]).(*RPCClient)
	} else {
		rpcConfig := config.RootSection("unittest")
		InitConfig(rpcConfig)
		rpcConfig.Set(ConfigBatchEnabled, true)
		rpcOptions := ReadConfig(ctx, rpcConfig)
		rb = NewRPCClientWithOption(c, rpcOptions).(*RPCClient)
	}

	return ctx, rb, func() {
		cancelCtx()
		server.Close()
	}
}

func TestNewRPCClientFailDueToBatchContextMissing(t *testing.T) {

	rpcConfig := config.RootSection("unittest")
	InitConfig(rpcConfig)
	rpcConfig.Set(ConfigBatchEnabled, true)

	assert.Panics(t, func() {
		NewRPCClientWithOption(nil, ReadConfig(nil, rpcConfig))
	})
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

	logrus.SetLevel(logrus.TraceLevel)

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
	assert.Empty(t, err)
	assert.Equal(t, int64(0x26), txCount.BigInt().Int64())
}

func TestSyncRPCCallNullResponse(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionReceipt", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`, rpcReq.Params[0].String())
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  nil,
		}
	})
	rb.requestCounter = 12345
	defer done()

	rpcRes, err := rb.SyncRequest(ctx, &RPCRequest{
		ID:     fftypes.JSONAnyPtr("1"),
		Method: "eth_getTransactionReceipt",
		Params: []*fftypes.JSONAny{
			fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, `null`, rpcRes.Result.String())
}

func TestSyncRPCCallErrorResponse(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 500, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Error: &RPCError{
				Message: "pop",
			},
		}
	})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "pop", err)
}

func TestSyncRPCCallBadJSONResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{!!!!`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClient(c).(*RPCClient)

	var txCount ethtypes.HexInteger
	rpcErr := rb.CallRPC(context.Background(), &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22012", rpcErr.Error())
}

func TestSyncRPCCallFailParseJSONResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"not an object"}`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClient(c).(*RPCClient)

	var mapResult map[string]interface{}
	rpcErr := rb.CallRPC(context.Background(), &mapResult, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22065", rpcErr.Error())
}

func TestSyncRPCCallErrorBadInput(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "test-bad-params", map[bool]bool{false: true})
	assert.Regexp(t, "FF22011", err)
}

func TestSyncRPCCallServerDown(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "net_version")
	assert.Regexp(t, "FF22012", err)
}

func TestSafeMessageGetter(t *testing.T) {

	assert.Empty(t, (&RPCResponse{}).Message())
}

func TestSyncRequestConcurrency(t *testing.T) {

	blocked := make(chan struct{})
	blocking := make(chan bool, 1)
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blocking <- true
		<-blocked
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{
		MaxConcurrentRequest: 1,
	}).(*RPCClient)

	bgDone := make(chan struct{})
	go func() {
		_, err := rb.SyncRequest(ctx, &RPCRequest{})
		assert.Regexp(t, "FF22012", err)
		close(bgDone)
	}()
	<-blocking

	cancelCtx()
	_, err = rb.SyncRequest(ctx, &RPCRequest{})
	assert.Regexp(t, "FF22063", err)

	close(blocked)
	<-bgDone
}

func TestBatchSyncRPCCallNullResponse(t *testing.T) {

	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (status int, rpcRes []*RPCResponse) {
		rpcReq := rpcReqs[0]
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionReceipt", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`, rpcReq.Params[0].String())
		return 200, []*RPCResponse{{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  nil}}
	})
	rb.requestCounter = 12345
	defer done()

	rpcRes, err := rb.SyncRequest(ctx, &RPCRequest{
		ID:     fftypes.JSONAnyPtr("1"),
		Method: "eth_getTransactionReceipt",
		Params: []*fftypes.JSONAny{
			fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, `null`, rpcRes.Result.String())
}

func TestBatchSyncRequestCanceledContext(t *testing.T) {

	blocked := make(chan struct{})
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
		cancelCtx()
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`[{}]`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{
		BatchOptions: &RPCClientBatchOptions{
			BatchDispatcherContext:      ctx,
			Enabled:                     true,
			BatchSize:                   1,
			BatchMaxDispatchConcurrency: 1,
		},
	}).(*RPCClient)

	checkDone := make(chan bool)
	go func() {
		_, err = rb.SyncRequest(ctx, &RPCRequest{})
		assert.Regexp(t, "FF22063", err) // this checks the response hit cancel context
		close(checkDone)
	}()
	close(blocked)
	<-checkDone

	// this checks request hit cancel context
	_, err = rb.SyncRequest(ctx, &RPCRequest{})
	assert.Regexp(t, "FF22063", err)
}

func TestBatchSyncRequestCanceledContextWhenQueueing(t *testing.T) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.FailNow(t, "Not expecting JSON-RPC endpoint to get called")
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	rpcConfig := config.RootSection("unittest")
	InitConfig(rpcConfig)
	rpcConfig.Set(ConfigBatchEnabled, true)
	rpcConfig.Set(ConfigBatchTimeout, "2h") // very long delay
	rpcConfig.Set(ConfigBatchSize, 2000)    // very big batch size

	rpcOptions := ReadConfig(ctx, rpcConfig)

	rb := NewRPCClientWithOption(c, rpcOptions).(*RPCClient)

	checkDone := make(chan bool)
	go func() {
		reqContext := context.Background()
		_, err = rb.SyncRequest(reqContext, &RPCRequest{})
		assert.Regexp(t, "FF22063", err) // this checks the response hit cancel context
		close(checkDone)
	}()
	time.Sleep(50 * time.Millisecond) // wait for the request to be queued and start the ticker
	cancelCtx()
	<-checkDone

	ctx2, cancelCtx2 := context.WithCancel(context.Background())

	rpcOptions = ReadConfig(ctx2, rpcConfig)
	rb = NewRPCClientWithOption(c, rpcOptions).(*RPCClient)

	checkDone = make(chan bool)
	go func() {
		reqContext := context.Background()
		_, err = rb.SyncRequest(reqContext, &RPCRequest{})
		assert.Regexp(t, "FF22063", err) // this checks the response hit cancel context
		close(checkDone)
	}()
	cancelCtx2() // cancel context straight away to check the pending request are drained correctly
	<-checkDone
}

func TestBatchSyncRequestCanceledContextWhenDispatchingABatch(t *testing.T) {

	blocked := make(chan struct{})
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`[{}]`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{
		BatchOptions: &RPCClientBatchOptions{
			BatchDispatcherContext:      ctx,
			Enabled:                     true,
			BatchSize:                   1,
			BatchMaxDispatchConcurrency: 1,
		},
	}).(*RPCClient)

	rb.requestBatchConcurrencySlots <- true // fill the worker slot so all batch will be queueing

	checkDone := make(chan bool)
	go func() {
		_, err = rb.SyncRequest(ctx, &RPCRequest{})
		assert.Regexp(t, "FF22063", err) // this checks the response hit cancel context
		close(checkDone)
	}()
	time.Sleep(50 * time.Millisecond) // wait for the quest to be queued in the other go routine
	cancelCtx()
	<-checkDone

}
func TestBatchSyncRPCCallErrorResponse(t *testing.T) {

	bgCtx := context.Background()

	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (status int, rpcRes []*RPCResponse) {
		assert.Equal(t, 1, len(rpcReqs))
		return 500, []*RPCResponse{{
			JSONRpc: "2.0",
			ID:      rpcReqs[0].ID,
			Error: &RPCError{
				Message: "pop",
			},
		}}
	}, RPCClientOptions{BatchOptions: &RPCClientBatchOptions{
		BatchDispatcherContext: bgCtx,
		Enabled:                true,
	}})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "pop", err)
}

func TestBatchSyncRPCCallErrorCountMismatch(t *testing.T) {
	bgCtx := context.Background()

	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (status int, rpcRes []*RPCResponse) {
		assert.Equal(t, 1, len(rpcReqs))
		return 500, []*RPCResponse{}
	}, RPCClientOptions{BatchOptions: &RPCClientBatchOptions{
		BatchDispatcherContext: bgCtx,
		Enabled:                true,
	}})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22087", err)
}

func TestBatchSyncRPCCallBadJSONResponse(t *testing.T) {
	bgCtx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{!!!!`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{BatchOptions: &RPCClientBatchOptions{
		BatchDispatcherContext: bgCtx,
		Enabled:                true,
	}}).(*RPCClient)

	var txCount ethtypes.HexInteger
	rpcErr := rb.CallRPC(context.Background(), &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22012", rpcErr.Error())
}

func TestBatchSyncRPCCallFailParseJSONResponse(t *testing.T) {
	bgCtx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`[{"result":"not an object"}]`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{BatchOptions: &RPCClientBatchOptions{
		BatchDispatcherContext: bgCtx,
		Enabled:                true,
	}}).(*RPCClient)

	var mapResult map[string]interface{}
	rpcErr := rb.CallRPC(context.Background(), &mapResult, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22065", rpcErr.Error())
}

func TestBatchSyncRPCCallErrorBadInput(t *testing.T) {

	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (status int, rpcRes []*RPCResponse) { return 500, nil })
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "test-bad-params", map[bool]bool{false: true})
	assert.Regexp(t, "FF22011", err)
}

func TestBatchRequestsOKWithBatchSize(t *testing.T) {
	// Define the expected server response to the batch
	rpcServerResponseBatchBytes := []byte(`[
		{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x24"
			}
		},
		{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x10"
			}
		}
	]`)

	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReqs []*RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReqs)
		assert.NoError(t, err)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(rpcServerResponseBatchBytes)))
		w.WriteHeader(200)
		w.Write(rpcServerResponseBatchBytes)
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rpcConfig := config.RootSection("unittest")
	InitConfig(rpcConfig)
	rpcConfig.Set(ConfigBatchEnabled, true)
	rpcConfig.Set(ConfigMaxConcurrentRequest, 10)
	rpcConfig.Set(ConfigBatchTimeout, "2h") // very long delay, so need to rely on batch size to be hit for sending a batch
	rpcConfig.Set(ConfigBatchSize, 2)

	rpcOptions := ReadConfig(ctx, rpcConfig)

	rb := NewRPCClientWithOption(c, rpcOptions).(*RPCClient)

	round := 400

	reqNumbers := 2*round - 1

	requestCount := make(chan bool, reqNumbers)

	for i := 0; i < reqNumbers; i++ {
		go func() {
			_, err := rb.SyncRequest(ctx, &RPCRequest{
				Method: "eth_getTransactionByHash",
				Params: []*fftypes.JSONAny{fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`)},
			})
			assert.Nil(t, err)
			requestCount <- true

		}()
	}

	for i := 0; i < reqNumbers-1; i++ {
		<-requestCount
	}

	_, err = rb.SyncRequest(ctx, &RPCRequest{
		Method: "eth_getTransactionByHash",
		Params: []*fftypes.JSONAny{fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`)},
	})
	assert.Nil(t, err)

	<-requestCount
}

func TestBatchRequestsTestWorkerCounts(t *testing.T) {
	// Define the expected server response to the batch
	rpcServerResponseBatchBytes := []byte(`[
		{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x24"
			}
		},
		{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x10"
			}
		}
	]`)

	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReqs []*RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReqs)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond) // set 200s delay for each quest
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(rpcServerResponseBatchBytes)))
		w.WriteHeader(200)
		w.Write(rpcServerResponseBatchBytes)
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	// only a single worker

	rpcConfig := config.RootSection("unittest")
	InitConfig(rpcConfig)
	rpcConfig.Set(ConfigBatchEnabled, true)
	rpcConfig.Set(ConfigMaxConcurrentRequest, 10)
	rpcConfig.Set(ConfigBatchTimeout, "2h") // very long delay, so need to rely on batch size to be hit for sending a batch
	rpcConfig.Set(ConfigBatchSize, 2)
	rpcConfig.Set(ConfigBatchMaxDispatchConcurrency, 1)

	rpcOptions := ReadConfig(ctx, rpcConfig)

	rb := NewRPCClientWithOption(c, rpcOptions).(*RPCClient)

	round := 5 // doing first round, each round will have at least 200ms delay, so the whole flow will guaranteed to be more than 1s

	reqNumbers := 2 * round

	requestCount := make(chan bool, reqNumbers)

	requestStart := time.Now()

	for i := 0; i < reqNumbers; i++ {
		go func() {
			_, err := rb.SyncRequest(ctx, &RPCRequest{
				ID:     fftypes.JSONAnyPtr("testId"),
				Method: "eth_getTransactionByHash",
				Params: []*fftypes.JSONAny{fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`)},
			})
			assert.Nil(t, err)
			requestCount <- true

		}()
	}

	for i := 0; i < reqNumbers; i++ {
		<-requestCount
	}

	assert.Greater(t, time.Since(requestStart), 1*time.Second)

	// number of worker equal to the number of rounds
	// so the delay should be slightly greater than per request delay (200ms), but hopefully less than 300ms (with 100ms overhead)
	rb = NewRPCClientWithOption(c, RPCClientOptions{
		MaxConcurrentRequest: 10,
		BatchOptions: &RPCClientBatchOptions{
			BatchDispatcherContext:      ctx,
			Enabled:                     true,
			BatchTimeout:                2 * time.Hour, // very long delay, so need to rely on batch size to be hit for sending a batch
			BatchSize:                   2,
			BatchMaxDispatchConcurrency: round,
		},
	}).(*RPCClient)

	requestStart = time.Now()

	for i := 0; i < reqNumbers; i++ {
		go func() {
			_, err := rb.SyncRequest(ctx, &RPCRequest{
				Method: "eth_getTransactionByHash",
				Params: []*fftypes.JSONAny{fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`)},
			})
			assert.Nil(t, err)
			requestCount <- true

		}()
	}

	for i := 0; i < reqNumbers; i++ {
		<-requestCount
	}

	assert.Greater(t, time.Since(requestStart), 200*time.Millisecond)
	assert.Less(t, time.Since(requestStart), 300*time.Millisecond)

}

func TestBatchRequestsOKWithBatchDelay(t *testing.T) {
	// Define the expected server response to the batch
	rpcServerResponseBatchBytes := []byte(`[
		{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x24"
			}
		},
		{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
				"nonce": "0x10"
			}
		}
	]`)

	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReqs []*RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReqs)
		assert.NoError(t, err)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(rpcServerResponseBatchBytes)))
		w.WriteHeader(200)
		w.Write(rpcServerResponseBatchBytes)
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	rpcConfig := config.RootSection("ut_fs_config")
	InitConfig(rpcConfig)
	rpcConfig.Set(ConfigBatchEnabled, true)
	rpcConfig.Set(ConfigMaxConcurrentRequest, 10)
	rpcConfig.Set(ConfigBatchTimeout, "100ms") // very long delay, so need to rely on batch size to be hit for sending a batch
	rpcConfig.Set(ConfigBatchSize, 2000)
	rpcConfig.Set(ConfigBatchMaxDispatchConcurrency, 1)

	rpcOptions := ReadConfig(ctx, rpcConfig)

	rb := NewRPCClientWithOption(c, rpcOptions).(*RPCClient)

	round := 5

	reqPerRound := 2

	for i := 0; i < round; i++ {
		requestCount := make(chan bool, reqPerRound)
		for i := 0; i < reqPerRound; i++ {
			go func() {
				_, err := rb.SyncRequest(ctx, &RPCRequest{
					Method: "eth_getTransactionByHash",
					Params: []*fftypes.JSONAny{fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`)},
				})
				assert.Nil(t, err)
				requestCount <- true

			}()
		}

		for i := 0; i < reqPerRound; i++ {
			<-requestCount
		}

	}

}
