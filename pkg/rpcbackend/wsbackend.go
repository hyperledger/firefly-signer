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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/sirupsen/logrus"
)

// WSBackend performs communication with a backend
type WSBackend interface {
	CallRPC(ctx context.Context, method string, params ...interface{}) (id string, rpcErr *RPCError)
	Subscribe(ctx context.Context, subChannel chan *RPCSubscriptionRequest, params ...interface{}) (error *RPCError)
	UnsubscribeAll(ctx context.Context) (error *RPCError)
	Connect(ctx context.Context) error
}

// NewRPCClient Constructor
func NewWSRPCClient(client wsclient.WSClient) WSBackend {
	return NewWSRPCClientWithOption(client, RPCClientOptions{})
}

// NewRPCClientWithOption Constructor
func NewWSRPCClientWithOption(client wsclient.WSClient, options RPCClientOptions) WSBackend {
	wsRPCClient := &WSRPCClient{
		client:               client,
		subscriptions:        make(map[string]chan *RPCSubscriptionRequest),
		pendingSubscriptions: make(map[string]chan *RPCSubscriptionRequest),
	}

	if options.MaxConcurrentRequest > 0 {
		wsRPCClient.concurrencySlots = make(chan bool, options.MaxConcurrentRequest)
	}

	return wsRPCClient
}

type WSRPCClient struct {
	client               wsclient.WSClient
	concurrencySlots     chan bool
	requestCounter       int64
	subscriptions        map[string]chan *RPCSubscriptionRequest
	pendingSubscriptions map[string]chan *RPCSubscriptionRequest
	pendingSubMutex      sync.Mutex
	subMutex             sync.Mutex
}

func (rc *WSRPCClient) Connect(ctx context.Context) error {
	if err := rc.client.Connect(); err != nil {
		return err
	}
	go rc.receiveLoop(ctx)
	return nil
}

func (rc *WSRPCClient) allocateRequestID(req *RPCRequest) string {
	reqID := fmt.Sprintf(`%.9d`, atomic.AddInt64(&rc.requestCounter, 1))
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	return reqID
}

func (rc *WSRPCClient) Subscribe(ctx context.Context, subChannel chan *RPCSubscriptionRequest, params ...interface{}) (error *RPCError) {
	rc.pendingSubMutex.Lock()
	defer rc.pendingSubMutex.Unlock()
	reqID, err := rc.CallRPC(ctx, "eth_subscribe", params...)
	if err != nil {
		return err
	}
	rc.pendingSubscriptions[reqID] = subChannel
	return nil
}

func (rc *WSRPCClient) UnsubscribeAll(ctx context.Context) (error *RPCError) {
	rc.subMutex.Lock()
	for subID, subChan := range rc.subscriptions {
		close(subChan)
		delete(rc.subscriptions, subID)
		_, err := rc.CallRPC(ctx, "eth_unsubscribe", subID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rc *WSRPCClient) CallRPC(ctx context.Context, method string, params ...interface{}) (id string, rpcErr *RPCError) {
	req := &RPCRequest{
		JSONRpc: "2.0",
		Method:  method,
		Params:  make([]*fftypes.JSONAny, len(params)),
	}
	for i, param := range params {
		b, err := json.Marshal(param)
		if err != nil {
			return "", &RPCError{Code: int64(RPCCodeInvalidRequest), Message: i18n.NewError(ctx, signermsgs.MsgInvalidParam, i, method, err).Error()}
		}
		req.Params[i] = fftypes.JSONAnyPtrBytes(b)
	}
	reqID, err := rc.request(ctx, req)
	if err != nil {
		return reqID, &RPCError{Code: int64(RPCCodeInvalidRequest), Message: i18n.NewError(ctx, signermsgs.MsgInvalidParam, 0, method, err).Error()}
	}
	return reqID, nil
}

func (rc *WSRPCClient) request(ctx context.Context, rpcReq *RPCRequest) (id string, err error) {
	if rc.concurrencySlots != nil {
		select {
		case rc.concurrencySlots <- true:
			// wait for the concurrency slot and continue
		case <-ctx.Done():
			return "", i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
		}
		defer func() {
			<-rc.concurrencySlots
		}()
	}

	// We always set the back-end request ID - as we need to support requests coming in from
	// multiple concurrent clients on our front-end that might use clashing IDs.
	reqID := rc.allocateRequestID(rpcReq)
	jsonInput, _ := json.Marshal(rpcReq)

	log.L(ctx).Debugf("RPC[%s] --> %s", reqID, rpcReq.Method)
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		log.L(ctx).Tracef("RPC[%s] INPUT: %s", reqID, jsonInput)
	}
	err = rc.client.Send(ctx, jsonInput)

	// Restore the original ID
	if err != nil {
		err := i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, err)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, err)
		return reqID, err
	}
	return reqID, nil
}

func (rc *WSRPCClient) receiveLoop(ctx context.Context) {
	for {
		bytes, ok := <-rc.client.Receive()
		if !ok {
			return
		}
		res := &RPCResponse{}
		if err := json.Unmarshal(bytes, res); err != nil {
			log.L(ctx).Errorf("RPC <-- ERROR: %s", err)
		}
		// If it doesn't have a result, it might be a request instead
		if res == nil || res.Result == nil || res.Result.String() == "" {
			req := &RPCSubscriptionRequest{}
			if err := json.Unmarshal(bytes, req); err != nil {
				log.L(ctx).Errorf("RPC <-- ERROR: %s", err)
			}
			// If it doesn't have a method I don't know what to do now
			if req == nil || req.Method == "" {
				log.L(ctx).Error("RPC <-- ERROR: Unable to process received message")
			}
			if req.Method == "eth_subscription" {
				subID := req.Params.Subscription.String()
				rc.subMutex.Lock()
				subChan, ok := rc.subscriptions[subID]
				rc.subMutex.Unlock()
				if ok {
					subChan <- req
				} else {
					// No active sub found for this one. Dropping it
					log.L(ctx).Warnf("RPC <-- WARN: Received subscription event for untracked subscription %s", subID)
				}
			}
		}
		rc.pendingSubMutex.Lock()
		id := res.ID.AsString()
		if subChan, ok := rc.pendingSubscriptions[id]; ok {
			delete(rc.pendingSubscriptions, res.ID.String())
			rc.pendingSubMutex.Unlock()
			subID := res.Result.AsString()
			rc.subMutex.Lock()
			rc.subscriptions[subID] = subChan
			rc.subMutex.Unlock()
		} else {
			rc.pendingSubMutex.Unlock()
		}
	}
}
