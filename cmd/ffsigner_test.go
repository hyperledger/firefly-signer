// Copyright © 2021 Kaleido, Inc.
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

package cmd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hyperledger/firefly-signer/mocks/rpcservermocks"
	"github.com/stretchr/testify/assert"
)

const configDir = "../test/data/config"

func TestRunOK(t *testing.T) {

	rootCmd.SetArgs([]string{"-f", "../test/firefly.ffsigner.yaml"})
	defer rootCmd.SetArgs([]string{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Execute()
		if err != nil {
			assert.Regexp(t, "context deadline", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)
	sigs <- os.Kill

	<-done

}

func TestRunNoWallet(t *testing.T) {

	rootCmd.SetArgs([]string{"-f", "../test/no-wallet.ffsigner.yaml"})
	defer rootCmd.SetArgs([]string{})

	err := Execute()
	assert.Regexp(t, "FF20217", err)

}

func TestRunBadConfig(t *testing.T) {

	rootCmd.SetArgs([]string{"-f", "../test/bad-config.ffsigner.yaml"})
	defer rootCmd.SetArgs([]string{})

	err := Execute()
	assert.Regexp(t, "FF00101", err)

}

func TestRunBadWalletConfig(t *testing.T) {

	rootCmd.SetArgs([]string{"-f", "../test/bad-wallet.ffsigner.yaml"})
	defer rootCmd.SetArgs([]string{})

	err := Execute()
	assert.Regexp(t, "FF20216", err)

}

func TestRunFailStartup(t *testing.T) {
	rootCmd.SetArgs([]string{"-f", "../test/quick-fail.ffsigner.yaml"})
	defer rootCmd.SetArgs([]string{})

	err := Execute()
	assert.Regexp(t, "FF10104", err)

}

func TestRunFailServer(t *testing.T) {

	s := &rpcservermocks.Server{}
	s.On("Start").Return(fmt.Errorf("pop"))
	err := runServer(s)
	assert.Regexp(t, err, "pop")

}
