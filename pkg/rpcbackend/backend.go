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
	"regexp"
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

	if options.BatchOptions != nil && options.BatchOptions.Enabled {
		if options.BatchOptions.BatchDispatcherContext == nil {
			panic("must provide a batch dispatcher context when batch is enabled")
		}
		batchTimeout := 50 * time.Millisecond
		batchSize := 500
		batchWorkerCount := 50
		if options.BatchOptions.BatchTimeout > 0 {
			batchTimeout = options.BatchOptions.BatchTimeout
		}

		if options.BatchOptions.BatchSize > 0 {
			batchSize = options.BatchOptions.BatchSize
		}
		if options.BatchOptions.BatchMaxDispatchConcurrency > 0 {
			batchWorkerCount = options.BatchOptions.BatchMaxDispatchConcurrency
		}

		if options.BatchOptions.BatchExcludeMethodsRegex != "" {
			excludeRegex, err := regexp.Compile(options.BatchOptions.BatchExcludeMethodsRegex)
			if err != nil {
				panic(err)
			}
			rpcClient.batchExcludeMethodsMatcher = excludeRegex
		}

		rpcClient.requestBatchConcurrencySlots = make(chan bool, batchWorkerCount)
		rpcClient.startBatchDispatcher(options.BatchOptions.BatchDispatcherContext, batchTimeout, batchSize)
	}

	return rpcClient
}

type RPCClient struct {
	client                       *resty.Client
	batchDispatcherContext       context.Context
	concurrencySlots             chan bool
	requestCounter               int64
	requestBatchQueue            chan *batchRequest
	requestBatchConcurrencySlots chan bool
	batchExcludeMethodsMatcher   *regexp.Regexp
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

type batchRequest struct {
	rpcReq *RPCRequest
	rpcRes chan *RPCResponse
	rpcErr chan error
}

func (rc *RPCClient) startBatchDispatcher(dispatcherRootContext context.Context, batchTimeout time.Duration, batchSize int) {
	rc.batchDispatcherContext = dispatcherRootContext
	if rc.requestBatchQueue == nil { // avoid orphaned dispatcher
		requestQueue := make(chan *batchRequest)
		go func() {
			var batch []*batchRequest
			var timeoutChannel <-chan time.Time
			for {
				select {
				case req := <-requestQueue:
					batch = append(batch, req)
					if timeoutChannel == nil {
						// first request received, start a batch timeout
						timeoutChannel = time.After(batchTimeout)
					}

					if len(batch) >= batchSize {
						rc.dispatchBatch(rc.batchDispatcherContext, batch)
						batch = nil
						timeoutChannel = nil // stop the timeout and let it get reset by the next request
					}

				case <-timeoutChannel:
					if len(batch) > 0 {
						rc.dispatchBatch(rc.batchDispatcherContext, batch)
						batch = nil
						timeoutChannel = nil // stop the timeout and let it get reset by the next request
					}
				case <-rc.batchDispatcherContext.Done():
					select { // drain the queue
					case req := <-requestQueue:
						batch = append(batch, req)
					default:
					}
					for i, req := range batch {
						// mark all queueing requests as failed
						cancelCtxErr := i18n.NewError(rc.batchDispatcherContext, signermsgs.MsgRequestCanceledContext, req.rpcReq.ID)
						batch[i].rpcErr <- cancelCtxErr
					}

					return
				}
			}
		}()
		rc.requestBatchQueue = requestQueue
	}
}

func (rc *RPCClient) dispatchBatch(ctx context.Context, batch []*batchRequest) {
	select {
	case rc.requestBatchConcurrencySlots <- true: // use a buffered channel to control the number of concurrent thread for batch dispatcher
		// wait for the worker slot and continue
	case <-ctx.Done():
		for _, req := range batch {
			err := i18n.NewError(ctx, signermsgs.MsgRPCRequestBatchFailed)
			req.rpcErr <- err
		}
		return
	}
	go func() {
		defer func() {
			<-rc.requestBatchConcurrencySlots
		}()

		// once a concurrency slot is obtained, dispatch the batch
		rc.dispatch(ctx, batch)
	}()
}

func (rc *RPCClient) dispatch(ctx context.Context, batch []*batchRequest) {

	batchRPCTraceID := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	traceIDs := make([]string, len(batch))

	var rpcReqs []*RPCRequest
	for i, req := range batch {
		// We always set the back-end request ID - as we need to support requests coming in from
		// multiple concurrent clients on our front-end that might use clashing IDs.
		var beReq = *req.rpcReq
		beReq.JSONRpc = "2.0"
		rpcTraceID := rc.allocateRequestID(&beReq)
		if req.rpcReq.ID != nil {
			// We're proxying a request with front-end RPC ID - log that as well
			rpcTraceID = fmt.Sprintf("%s->%s/%s", req.rpcReq.ID, batchRPCTraceID, rpcTraceID)
		}
		traceIDs[i] = rpcTraceID
		rpcReqs = append(rpcReqs, &beReq)
	}
	log.L(ctx).Debugf("RPC[%s] --> BATCH %d requests", batchRPCTraceID, len(rpcReqs))

	responses := make([]*RPCResponse, len(batch))
	res, err := rc.client.R().
		SetContext(ctx).
		SetBody(rpcReqs).
		SetResult(&responses).
		SetError(&responses).
		Post("")

	if err != nil {
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", batchRPCTraceID, err)
		for _, req := range batch {
			req.rpcErr <- err
		}
		return
	}

	if len(responses) != len(batch) {
		err := i18n.NewError(ctx, signermsgs.MsgRPCRequestBatchFailed)
		for _, req := range batch {
			req.rpcErr <- err
		}
		return
	}

	for i, resp := range responses {
		if logrus.IsLevelEnabled(logrus.TraceLevel) {
			jsonOutput, _ := json.Marshal(resp)
			log.L(ctx).Tracef("RPC[%s] OUTPUT: %s", batchRPCTraceID, jsonOutput)
		}

		// JSON/RPC allows errors to be returned with a 200 status code, as well as other status codes
		if res.IsError() || (resp != nil && resp.Error != nil && resp.Error.Code != 0) {
			rpcMsg := ""
			errLog := ""
			if resp != nil {
				rpcMsg = resp.Message()
				errLog = rpcMsg
			}
			if rpcMsg == "" {
				// Log the raw result in the case of JSON parse error etc. (note that Resty no longer
				// returns this as an error - rather the body comes back raw)
				errLog = string(res.Body())
				rpcMsg = i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, res.Status()).Error()
			}
			traceID := traceIDs[i]
			log.L(ctx).Errorf("RPC[%s] <-- [%d]: %s", traceID, res.StatusCode(), errLog)
			batch[i].rpcErr <- fmt.Errorf(rpcMsg)
		} else {
			if resp == nil {
				// .... sometimes the JSON-RPC endpoint could return null...
				resp = new(RPCResponse)
			}
			if resp.Result == nil {
				// We don't want a result for errors, but a null success response needs to go in there
				resp.Result = fftypes.JSONAnyPtr(fftypes.NullString)
			}
			batch[i].rpcRes <- resp

		}
	}
}

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

	if rc.requestBatchQueue != nil && (rc.batchExcludeMethodsMatcher == nil || !rc.batchExcludeMethodsMatcher.MatchString(rpcReq.Method)) {
		return rc.batchSyncRequest(ctx, rpcReq)
	} else {
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
			SetError(&rpcRes).
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

}

func (rc *RPCClient) batchSyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error) {
	req := &batchRequest{
		rpcReq: rpcReq,
		rpcRes: make(chan *RPCResponse, 1),
		rpcErr: make(chan error, 1),
	}

	select {
	case rc.requestBatchQueue <- req:
	case <-rc.batchDispatcherContext.Done():
		err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
		return RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError), err
	case <-ctx.Done():
		err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
		return RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError), err
	}

	select {
	case rpcRes := <-req.rpcRes:
		return rpcRes, nil
	case err := <-req.rpcErr:
		return nil, err
	case <-rc.batchDispatcherContext.Done():
		err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
		return RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError), err
	case <-ctx.Done():
		err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, rpcReq.ID)
		return RPCErrorResponse(err, rpcReq.ID, RPCCodeInternalError), err
	}
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
