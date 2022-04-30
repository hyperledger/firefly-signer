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
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/mocks/ethsignermocks"
	"github.com/hyperledger/firefly-signer/mocks/rpcbackendmocks"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly/pkg/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestServer(t *testing.T) (string, *rpcServer, func()) {
	signerconfig.Reset()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	serverPort := strings.Split(ln.Addr().String(), ":")[1]
	ln.Close()
	signerconfig.ServerPrefix.Set(httpserver.HTTPConfPort, serverPort)
	signerconfig.ServerPrefix.Set(httpserver.HTTPConfAddress, "127.0.0.1")

	w := &ethsignermocks.Wallet{}

	ss, err := NewServer(context.Background(), w)
	assert.NoError(t, err)
	s := ss.(*rpcServer)
	s.backend = &rpcbackendmocks.Backend{}

	return fmt.Sprintf("http://127.0.0.1:%s", serverPort),
		s,
		func() {
			s.Stop()
			_ = s.WaitStop()
		}

}

func TestStartStop(t *testing.T) {

	_, s, done := newTestServer(t)
	defer done()

	bm := s.backend.(*rpcbackendmocks.Backend)
	bm.On("CallRPC", mock.Anything, mock.Anything, "net_version").Run(func(args mock.Arguments) {
		hi := args[1].(*ethtypes.HexInteger)
		hi.BigInt().SetInt64(12345)
	}).Return(nil)

	w := s.wallet.(*ethsignermocks.Wallet)
	w.On("Initialize", mock.Anything).Return(nil)
	err := s.Start()
	assert.NoError(t, err)

	assert.Equal(t, int64(12345), s.chainID)

}

func TestStartFailChainID(t *testing.T) {

	_, s, done := newTestServer(t)
	defer done()

	bm := s.backend.(*rpcbackendmocks.Backend)
	bm.On("CallRPC", mock.Anything, mock.Anything, "net_version").Run(func(args mock.Arguments) {
		hi := args[1].(*ethtypes.HexInteger)
		hi.BigInt().SetInt64(12345)
	}).Return(fmt.Errorf("pop"))

	err := s.Start()
	assert.Regexp(t, "pop", err)

}

func TestStartFailInitialize(t *testing.T) {

	_, s, done := newTestServer(t)
	defer done()

	bm := s.backend.(*rpcbackendmocks.Backend)
	bm.On("CallRPC", mock.Anything, mock.Anything, "net_version").Run(func(args mock.Arguments) {
		hi := args[1].(*ethtypes.HexInteger)
		hi.BigInt().SetInt64(12345)
	}).Return(nil)

	w := s.wallet.(*ethsignermocks.Wallet)
	w.On("Initialize", mock.Anything).Return(fmt.Errorf("pop"))
	err := s.Start()
	assert.Regexp(t, "pop", err)

}

func TestBadConfig(t *testing.T) {

	signerconfig.Reset()
	signerconfig.ServerPrefix.Set(httpserver.HTTPConfAddress, ":::::")
	_, err := NewServer(context.Background(), &ethsignermocks.Wallet{})
	assert.Error(t, err)

}
