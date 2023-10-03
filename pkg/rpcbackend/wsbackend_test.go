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

package rpcbackend

import (
	"context"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

func generateConfig() *wsclient.WSConfig {
	return &wsclient.WSConfig{}
}

func TestWSRPCConnect(t *testing.T) {
	_, _, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})
	defer close()

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.NoError(t, err)
}

func TestWSRPCConnectError(t *testing.T) {
	// Init clean config
	wsConfig := generateConfig()

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.Regexp(t, "FF00148", err)
}

func TestWSRPCSubscribe(t *testing.T) {
	toServer, fromServer, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})
	defer close()

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.NoError(t, err)

	subChan := make(chan *RPCSubscriptionRequest)
	wsRPCClient.Subscribe(context.Background(), subChan, "newHeads")

	msg := <-toServer
	assert.Equal(t, "{\"jsonrpc\":\"2.0\",\"id\":\"000000001\",\"method\":\"eth_subscribe\",\"params\":[\"newHeads\"]}", msg)

	// Test error cases first to make sure client ignores stuff it doesn't care about
	// should log: WARN: Received subscription event for untracked subscription
	fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x99999999999999999999999999999999"}}`
	// should log: ERROR: Unable to process received message
	fromServer <- `{"nonsense": true}`
	// should log a deserialization error
	fromServer <- `notjson`

	// Then test real subscription message
	fromServer <- `{"jsonrpc":"2.0","id":"000000001","result":"0x9ce59a13059e417087c02d3236a0b1cc"}`
	fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x9ce59a13059e417087c02d3236a0b1cc"}}`

	newHead := <-subChan
	assert.NotNil(t, newHead)

	blockNumber := ethtypes.NewHexInteger(newHead.Params.Result.JSONObject().GetInteger("number"))
	assert.Equal(t, big.NewInt(1263817), blockNumber.BigInt())

	hash := newHead.Params.Result.JSONObject().GetString("hash")
	assert.Equal(t, "0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378", hash)

	wsRPCClient.UnsubscribeAll(context.Background())

	res, ok := <-subChan
	assert.Nil(t, res)
	assert.False(t, ok)

	wsRPCClient.Close()
}

func TestWSRPCSubscribeError(t *testing.T) {
	_, _, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})
	defer close()

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	subChan := make(chan *RPCSubscriptionRequest)
	wsRPCClient.Subscribe(ctx, subChan, []bool{false})
}

func TestWSRPCCallRPCError(t *testing.T) {
	_, _, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})
	defer close()

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.NoError(t, err)

	bad := map[bool]bool{false: true}
	_, rpcErr := wsRPCClient.CallRPC(context.Background(), "eth_call", bad)
	assert.Error(t, rpcErr.Error())
}

func TestWSRPCUnsubscribeError(t *testing.T) {
	toServer, fromServer, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})
	defer close()

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2

	wsc, err := wsclient.New(context.Background(), wsConfig, nil, nil)
	assert.NoError(t, err)

	wsRPCClient := NewWSRPCClient(wsc)

	err = wsRPCClient.Connect(context.Background())
	assert.NoError(t, err)

	subChan := make(chan *RPCSubscriptionRequest)
	wsRPCClient.Subscribe(context.Background(), subChan, "newHeads")

	msg := <-toServer
	assert.Equal(t, "{\"jsonrpc\":\"2.0\",\"id\":\"000000001\",\"method\":\"eth_subscribe\",\"params\":[\"newHeads\"]}", msg)
	fromServer <- `{"jsonrpc":"2.0","id":"000000001","result":"0x9ce59a13059e417087c02d3236a0b1cc"}`
	fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x9ce59a13059e417087c02d3236a0b1cc"}}`

	newHead := <-subChan
	assert.NotNil(t, newHead)

	blockNumber := ethtypes.NewHexInteger(newHead.Params.Result.JSONObject().GetInteger("number"))
	assert.Equal(t, big.NewInt(1263817), blockNumber.BigInt())

	hash := newHead.Params.Result.JSONObject().GetString("hash")
	assert.Equal(t, "0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378", hash)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wsRPCClient.UnsubscribeAll(ctx)

	res, ok := <-subChan
	assert.Nil(t, res)
	assert.False(t, ok)

	wsRPCClient.Close()
}
