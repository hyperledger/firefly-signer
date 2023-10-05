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
	"time"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/sirupsen/logrus"
)

// WSBackend performs communication with a backend
type WSBackend interface {
	RPCCaller
	Subscribe(ctx context.Context, params ...interface{}) (sub Subscription, error *RPCError)
	UnsubscribeAll(ctx context.Context) (error *RPCError)
	Connect(ctx context.Context) error
	Close()
}

// NewRPCClient Constructor
func NewWSRPCClient(client wsclient.WSClient) WSBackend {
	return &wsRPCClient{
		client:        client,
		calls:         make(map[string]chan *RPCResponse),
		pendingSubs:   make(map[string]chan *newSubResponse),
		subscriptions: make(map[string]*sub),
	}
}

type Subscription interface {
	ID() string
	Notifications() chan *RPCSubscriptionNotification
	Unsubscribe(ctx context.Context) *RPCError
}

type RPCSubscriptionNotification struct {
	Subscription Subscription
	Result       *fftypes.JSONAny
}

type wsRPCClient struct {
	mux            sync.Mutex
	client         wsclient.WSClient
	requestCounter int64
	calls          map[string]chan *RPCResponse
	pendingSubs    map[string]chan *newSubResponse
	subscriptions  map[string]*sub
}

type sub struct {
	rc             *wsRPCClient
	ctx            context.Context
	cancelCtx      context.CancelFunc
	subscriptionID string
	notifications  chan *RPCSubscriptionNotification
}

type newSubResponse struct {
	s      *sub
	rpcErr *RPCError
}

func (rc *wsRPCClient) Connect(ctx context.Context) error {
	if err := rc.client.Connect(); err != nil {
		return err
	}
	go rc.receiveLoop(log.WithLogField(ctx, "role", "rpc_websocket"))
	return nil
}

func (rc *wsRPCClient) Close() {
	rc.client.Close()
}

func (rc *wsRPCClient) addInflightRequest(req *RPCRequest) (string, chan *RPCResponse) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	rc.requestCounter++
	reqID := fmt.Sprintf("%.9d", rc.requestCounter)
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	resChl := make(chan *RPCResponse, 1)
	rc.calls[reqID] = resChl
	return reqID, resChl
}

func (rc *wsRPCClient) addInflightSub(req *RPCRequest) (string, chan *newSubResponse) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	rc.requestCounter++
	reqID := fmt.Sprintf("%.9d", rc.requestCounter)
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	resChl := make(chan *newSubResponse, 1)
	rc.pendingSubs[reqID] = resChl
	return reqID, resChl
}

func (rc *wsRPCClient) popInflight(rpcID string) (chan *newSubResponse, chan *RPCResponse) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	inflightSub, ok := rc.pendingSubs[rpcID]
	if ok {
		delete(rc.pendingSubs, rpcID)
		return inflightSub, nil
	}
	inflightCall, ok := rc.calls[rpcID]
	if ok {
		delete(rc.calls, rpcID)
		return nil, inflightCall
	}
	return nil, nil
}

func (rc *wsRPCClient) addSubscription(s *sub) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	rc.subscriptions[s.subscriptionID] = s
}

func (rc *wsRPCClient) removeSubscription(s *sub) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	delete(rc.subscriptions, s.subscriptionID)
}

func (rc *wsRPCClient) getSubscription(subscriptionID string) *sub {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	return rc.subscriptions[subscriptionID]
}

func (rc *wsRPCClient) getAllSubscriptions() []*sub {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	subs := make([]*sub, 0, len(rc.subscriptions))
	for _, s := range rc.subscriptions {
		subs = append(subs, s)
	}
	return subs
}

func (rc *wsRPCClient) removeInflightRequest(reqID string) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	delete(rc.calls, reqID)
}

func (rc *wsRPCClient) Subscribe(ctx context.Context, params ...interface{}) (sub Subscription, error *RPCError) {

	// We can't just use RPCCall here, because we have to intercept the response differently on the routine
	rpcReq, rpcErr := buildRequest(ctx, "eth_subscribe", params)
	if rpcErr != nil {
		return nil, rpcErr
	}

	reqID, resChannel := rc.addInflightSub(rpcReq)
	defer rc.removeInflightRequest(reqID)

	if rpcErr = rc.sendRPC(ctx, reqID, rpcReq); rpcErr != nil {
		return nil, rpcErr
	}

	select {
	case nsr := <-resChannel:
		return nsr.s, nsr.rpcErr
	case <-ctx.Done():
		return nil, NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRequestCanceledContext, reqID)
	}
}

func (s *sub) ID() string {
	return s.subscriptionID
}

func (s *sub) Notifications() chan *RPCSubscriptionNotification {
	return s.notifications
}

func (s *sub) Unsubscribe(ctx context.Context) *RPCError {
	s.cancelCtx() // we unblock the receiver if it was previously trying to dispatch to this subscription, before invoking unsubscribe
	s.rc.removeSubscription(s)
	var resultBool bool
	rpcErr := s.rc.CallRPC(ctx, &resultBool, "eth_unsubscribe", s.subscriptionID)
	if rpcErr != nil {
		return rpcErr
	}
	log.L(ctx).Infof("Unsubscribed '%s' (result=%t)", s.subscriptionID, resultBool)
	close(s.notifications)
	return nil
}

func (rc *wsRPCClient) UnsubscribeAll(ctx context.Context) (lastErr *RPCError) {
	for _, s := range rc.getAllSubscriptions() {
		if lastErr = s.Unsubscribe(ctx); lastErr != nil {
			log.L(ctx).Errorf("Failed to unsubscribe %s: %s", s.subscriptionID, lastErr)
		}
	}
	return lastErr
}

func (rc *wsRPCClient) sendRPC(ctx context.Context, reqID string, rpcReq *RPCRequest) *RPCError {
	jsonInput, _ := json.Marshal(rpcReq)
	log.L(ctx).Debugf("RPC[%s] --> %s", reqID, rpcReq.Method)
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		log.L(ctx).Tracef("RPC[%s] INPUT: %s", reqID, jsonInput)
	}
	err := rc.client.Send(ctx, jsonInput)
	if err != nil {
		rpcErr := NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRPCRequestFailed, err)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, err)
		return rpcErr
	}
	return nil
}

func (rc *wsRPCClient) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError {
	rpcReq, rpcErr := buildRequest(ctx, method, params)
	if rpcErr != nil {
		return rpcErr
	}

	reqID, resChannel := rc.addInflightRequest(rpcReq)
	defer rc.removeInflightRequest(reqID)

	rpcStartTime := time.Now()
	if rpcErr = rc.sendRPC(ctx, reqID, rpcReq); rpcErr != nil {
		return rpcErr
	}
	return rc.waitResponse(ctx, result, reqID, rpcReq, rpcStartTime, resChannel)
}

func (rc *wsRPCClient) waitResponse(ctx context.Context, result interface{}, reqID string, rpcReq *RPCRequest, rpcStartTime time.Time, resChannel chan *RPCResponse) *RPCError {
	var rpcRes *RPCResponse
	select {
	case rpcRes = <-resChannel:
	case <-ctx.Done():
		rpcErr := NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRequestCanceledContext, reqID)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, rpcErr.Error())
		return rpcErr
	}
	if rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, rpcRes.Message())
		return rpcRes.Error
	}
	log.L(ctx).Infof("RPC[%s] <-- %s OK (%.2fms)", reqID, rpcReq.Method, float64(time.Since(rpcStartTime))/float64(time.Millisecond))
	if rpcRes.Result == nil {
		// We don't want a result for errors, but a null success response needs to go in there
		rpcRes.Result = fftypes.JSONAnyPtr(fftypes.NullString)
	}
	if result != nil {
		if err := json.Unmarshal(rpcRes.Result.Bytes(), &result); err != nil {
			err = i18n.NewError(ctx, signermsgs.MsgResultParseFailed, result, err)
			return &RPCError{Code: int64(RPCCodeParseError), Message: err.Error()}
		}
	}
	return nil
}

func (rc *wsRPCClient) handleSubscriptionNotification(ctx context.Context, rpcRes *RPCResponse) {
	type rpcSubscriptionParams struct {
		Subscription string           `json:"subscription"` // probably hex, but not protocol assured
		Result       *fftypes.JSONAny `json:"result,omitempty"`
	}
	var subParams rpcSubscriptionParams
	if rpcRes.Params != nil {
		_ = json.Unmarshal(rpcRes.Params.Bytes(), &subParams)
	}
	if len(subParams.Subscription) == 0 {
		log.L(ctx).Warnf("RPC[%s] <-- Unable to extract subscription id from notification: %s", rpcRes.ID.AsString(), rpcRes.Params)
		return
	}

	sub := rc.getSubscription(subParams.Subscription)
	if sub == nil {
		log.L(ctx).Warnf("RPC[%s] <-- Notification for unknown subscription '%s'", rpcRes.ID.AsString(), subParams.Subscription)
		return
	}

	// This is a notification that should match an active subscription
	select {
	case sub.notifications <- &RPCSubscriptionNotification{
		Subscription: sub,
		Result:       subParams.Result,
	}:
	case <-sub.ctx.Done():
		// The subscription has been unsubscribed, or we're closing
		log.L(ctx).Warnf("RPC[%s] <-- Received subscription event after unsubscribe/close %s", rpcRes.ID.AsString(), sub.subscriptionID)
	}
}

func (rc *wsRPCClient) handleSubscriptionConfirm(ctx context.Context, rpcRes *RPCResponse, inflightSubscribe chan *newSubResponse) {
	if rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		inflightSubscribe <- &newSubResponse{
			rpcErr: rpcRes.Error,
		}
		return
	}
	var subscriptionID string // we know it's probably hex, but we cannot rely on that being guaranteed
	if rpcRes.Result != nil {
		_ = json.Unmarshal(rpcRes.Result.Bytes(), &subscriptionID)
	}
	if len(subscriptionID) == 0 {
		log.L(ctx).Warnf("RPC[%s] <-- Unable to extract subscription id from eth_subscribe response: %s", rpcRes.ID.AsString(), rpcRes.Params)
		inflightSubscribe <- &newSubResponse{
			rpcErr: NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgSubscribeResponseInvalid),
		}
		return
	}
	s := &sub{
		rc:             rc,
		subscriptionID: subscriptionID,
		notifications:  make(chan *RPCSubscriptionNotification), // blocking channel for these, but Unsubscribe will unblock by cancelling ctx
	}
	s.ctx, s.cancelCtx = context.WithCancel(ctx)
	log.L(ctx).Infof("Subscribed '%s'", s.subscriptionID)
	rc.addSubscription(s)
	inflightSubscribe <- &newSubResponse{
		s: s,
	}
}

func (rc *wsRPCClient) receiveLoop(ctx context.Context) {
	for {
		bytes, ok := <-rc.client.Receive()
		if !ok {
			log.L(ctx).Debugf("WebSocket closed")
			return
		}
		rpcRes := RPCResponse{}
		if err := json.Unmarshal(bytes, &rpcRes); err != nil {
			log.L(ctx).Errorf("RPC <-- ERROR invalid data '%s': %s", bytes, err)
		} else if rpcRes.Method == "eth_subscription" {
			rc.handleSubscriptionNotification(ctx, &rpcRes)
		} else {
			// ID should match a request we sent
			inflightSubscribe, inflightCall := rc.popInflight(rpcRes.ID.AsString())
			switch {
			case inflightSubscribe != nil:
				rc.handleSubscriptionConfirm(ctx, &rpcRes, inflightSubscribe)
			case inflightCall != nil:
				inflightCall <- &rpcRes // assured not to block as we allocate one slot, and pop first time we see it
			default:
				log.L(ctx).Warnf("RPC[%s] <-- Received unexpected RPC response: %+v", rpcRes.ID.AsString(), rpcRes)
			}
		}
	}
}
