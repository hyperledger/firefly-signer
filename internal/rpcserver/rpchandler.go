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
	"io/ioutil"
	"net/http"
	"strconv"
	"unicode"

	"github.com/hyperledger/firefly-signer/internal/rpcbackend"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
)

func (s *rpcServer) rpcHandler(w http.ResponseWriter, r *http.Request) {

	// Add a UUID to all logs on this context
	ctx := log.WithLogField(r.Context(), "r", fftypes.NewUUID().String())

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.replyRPCParseError(ctx, w, b)
		return
	}

	if s.sniffFirstByte(b) == '[' {
		s.handleRPCBatch(ctx, w, b)
		return
	}

	var rpcRequest rpcbackend.RPCRequest
	err = json.Unmarshal(b, &rpcRequest)
	if err != nil {
		s.replyRPCParseError(ctx, w, b)
		return
	}
	rpcResponse, err := s.processRPC(ctx, &rpcRequest)
	if err != nil {
		s.replyRPC(w, rpcResponse, http.StatusInternalServerError)
		return
	}
	s.replyRPC(w, rpcResponse, http.StatusOK)

}

func (s *rpcServer) replyRPCParseError(ctx context.Context, w http.ResponseWriter, b []byte) {
	log.L(ctx).Errorf("Request could not be parsed: %s", b)
	rpcError := rpcbackend.RPCErrorResponse(
		i18n.NewError(ctx, signermsgs.MsgInvalidRequest),
		fftypes.JSONAnyPtr("1"), // we couldn't parse the request ID
		rpcbackend.RPCCodeInvalidRequest,
	)
	s.replyRPC(w, rpcError, http.StatusBadRequest)
}

func (s *rpcServer) replyRPC(w http.ResponseWriter, result interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(result)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func (s *rpcServer) sniffFirstByte(data []byte) byte {
	sniffLen := len(data)
	if sniffLen > 100 {
		sniffLen = 100
	}
	for _, b := range data[0:sniffLen] {
		if !unicode.IsSpace(rune(b)) {
			return b
		}
	}
	return 0x00
}

func (s *rpcServer) handleRPCBatch(ctx context.Context, w http.ResponseWriter, batchBytes []byte) {

	var rpcArray []*rpcbackend.RPCRequest
	err := json.Unmarshal(batchBytes, &rpcArray)
	if err != nil || len(rpcArray) == 0 {
		log.L(ctx).Errorf("Bad RPC array received %s", batchBytes)
		s.replyRPCParseError(ctx, w, batchBytes)
		return
	}

	// Kick off a routine to fill in each
	rpcResponses := make([]*rpcbackend.RPCResponse, len(rpcArray))
	results := make(chan error)
	for i, r := range rpcArray {
		responseNumber := i
		rpcReq := r
		go func() {
			var err error
			rpcResponses[responseNumber], err = s.processRPC(ctx, rpcReq)
			results <- err
		}()
	}
	status := 200
	for range rpcArray {
		err := <-results
		if err != nil {
			status = http.StatusInternalServerError
		}
	}
	s.replyRPC(w, rpcResponses, status)
}
