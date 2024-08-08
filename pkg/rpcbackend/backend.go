// Copyright © 2024 Kaleido, Inc.
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
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/sirupsen/logrus"
)

type RPCCode int64

const (
	RPCCodeParseError     RPCCode = -32700
	RPCCodeInvalidRequest RPCCode = -32600
	RPCCodeInternalError  RPCCode = -32603
)

type RPC interface {
	CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError
}

// Backend performs communication with a backend
type Backend interface {
	RPC
	SyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error)
}

// NewRPCClient Constructor
func NewRPCClient(client *resty.Client) Backend {
	return NewRPCClientWithOption(client, RPCClientOptions{})
}

// NewRPCClientWithOption Constructor
func NewRPCClientWithOption(client *resty.Client, options RPCClientOptions) Backend {
	rpcClient := &RPCClient{
		client: client,
	}

	if options.MaxConcurrentRequest > 0 {
		rpcClient.concurrencySlots = make(chan bool, options.MaxConcurrentRequest)
	}

	return rpcClient
}

type RPCClient struct {
	client           *resty.Client
	concurrencySlots chan bool
	requestCounter   int64
}

type RPCClientOptions struct {
	MaxConcurrentRequest int64
}

type RPCRequest struct {
	JSONRpc string             `json:"jsonrpc"`
	ID      *fftypes.JSONAny   `json:"id"`
	Method  string             `json:"method"`
	Params  []*fftypes.JSONAny `json:"params,omitempty"`
}

type RPCError struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    fftypes.JSONAny `json:"data,omitempty"`
}

func (e *RPCError) Error() error {
	return fmt.Errorf(e.Message)
}

func (e *RPCError) String() string {
	return e.Message
}

type RPCResponse struct {
	JSONRpc string           `json:"jsonrpc"`
	ID      *fftypes.JSONAny `json:"id"`
	Result  *fftypes.JSONAny `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
	// Only for subscription notifications
	Method string           `json:"method,omitempty"`
	Params *fftypes.JSONAny `json:"params,omitempty"`
}

func (r *RPCResponse) Message() string {
	if r.Error != nil {
		return r.Error.Message
	}
	return ""
}

func (rc *RPCClient) allocateRequestID(req *RPCRequest) string {
	reqID := fmt.Sprintf(`%.9d`, atomic.AddInt64(&rc.requestCounter, 1))
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	return reqID
}

func (rc *RPCClient) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError {
	rpcReq, rpcErr := buildRequest(ctx, method, params)
	if rpcErr != nil {
		return rpcErr
	}
	res, err := rc.SyncRequest(ctx, rpcReq)
	if err != nil {
		if res != nil && res.Error != nil && res.Error.Code != 0 {
			return res.Error
		}
		return &RPCError{Code: int64(RPCCodeInternalError), Message: err.Error()}
	}
	err = json.Unmarshal(res.Result.Bytes(), &result)
	if err != nil {
		err = i18n.NewError(ctx, signermsgs.MsgResultParseFailed, result, err)
		return &RPCError{Code: int64(RPCCodeParseError), Message: err.Error()}
	}
	return nil
}

// SyncRequest sends an individual RPC request to the backend (always over HTTP currently),
// and waits synchronously for the response, or an error.
//
// In all return paths *including error paths* the RPCResponse is populated
// so the caller has an RPC structure to send back to the front-end caller.
func (rc *RPCClient) SyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error) {
	if rc.concurrencySlots != nil {
		select {
		case rc.concurrencySlots <- true:
			// wait for the concurrency slot and continue
		case <-ctx.Done():
			err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
			return RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError), err
		}
		defer func() {
			<-rc.concurrencySlots
		}()
	}

	// We always set the back-end request ID - as we need to support requests coming in from
	// multiple concurrent clients on our front-end that might use clashing IDs.
	var beReq = *rpcReq
	beReq.JSONRpc = "2.0"
	rpcTraceID := rc.allocateRequestID(&beReq)
	if rpcReq.ID != nil {
		// We're proxying a request with front-end RPC ID - log that as well
		rpcTraceID = fmt.Sprintf("%s->%s", rpcReq.ID, rpcTraceID)
	}

	rpcRes = new(RPCResponse)

	log.L(ctx).Debugf("RPC[%s] --> %s", rpcTraceID, rpcReq.Method)
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		jsonInput, _ := json.Marshal(rpcReq)
		log.L(ctx).Tracef("RPC[%s] INPUT: %s", rpcTraceID, jsonInput)
	}
	rpcStartTime := time.Now()
	res, err := rc.client.R().
		SetContext(ctx).
		SetBody(beReq).
		SetResult(&rpcRes).
		SetError(rpcRes).
		Post("")

	// Restore the original ID
	rpcRes.ID = rpcReq.ID
	if err != nil {
		err := i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, err)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", rpcTraceID, err)
		rpcRes = RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError)
		return rpcRes, err
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		jsonOutput, _ := json.Marshal(rpcRes)
		log.L(ctx).Tracef("RPC[%s] OUTPUT: %s", rpcTraceID, jsonOutput)
	}
	// JSON/RPC allows errors to be returned with a 200 status code, as well as other status codes
	if res.IsError() || rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		rpcMsg := rpcRes.Message()
		errLog := rpcMsg
		if rpcMsg == "" {
			// Log the raw result in the case of JSON parse error etc. (note that Resty no longer
			// returns this as an error - rather the body comes back raw)
			errLog = string(res.Body())
			rpcMsg = i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, res.Status()).Error()
		}
		log.L(ctx).Errorf("RPC[%s] <-- [%d]: %s", rpcTraceID, res.StatusCode(), errLog)
		err := fmt.Errorf(rpcMsg)
		return rpcRes, err
	}
	log.L(ctx).Infof("RPC[%s] <-- %s [%d] OK (%.2fms)", rpcTraceID, rpcReq.Method, res.StatusCode(), float64(time.Since(rpcStartTime))/float64(time.Millisecond))
	if rpcRes.Result == nil {
		// We don't want a result for errors, but a null success response needs to go in there
		rpcRes.Result = fftypes.JSONAnyPtr(fftypes.NullString)
	}
	return rpcRes, nil
}

func RPCErrorResponse(err error, id *fftypes.JSONAny, code RPCCode) *RPCResponse {
	return &RPCResponse{
		JSONRpc: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    int64(code),
			Message: err.Error(),
		},
	}
}

func NewRPCError(ctx context.Context, code RPCCode, msg i18n.ErrorMessageKey, inserts ...interface{}) *RPCError {
	return &RPCError{Code: int64(code), Message: i18n.NewError(ctx, msg, inserts...).Error()}
}

func buildRequest(ctx context.Context, method string, params []interface{}) (*RPCRequest, *RPCError) {
	req := &RPCRequest{
		JSONRpc: "2.0",
		Method:  method,
		Params:  make([]*fftypes.JSONAny, len(params)),
	}
	for i, param := range params {
		b, err := json.Marshal(param)
		if err != nil {
			return nil, NewRPCError(ctx, RPCCodeInvalidRequest, signermsgs.MsgInvalidParam, i, method, err)
		}
		req.Params[i] = fftypes.JSONAnyPtrBytes(b)
	}
	return req, nil
}
