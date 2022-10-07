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

package fswallet

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newEmptyWalletTestDir(t *testing.T, init bool) (context.Context, *fsWallet, chan ethtypes.Address0xHex, func()) {
	config.RootConfigReset()
	logrus.SetLevel(logrus.TraceLevel)

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, t.TempDir())
	unitTestConfig.Set(ConfigFilenamesPrimaryMatchRegex, "^((0x)?[0-9a-z]+).key.json$")
	unitTestConfig.Set(ConfigFilenamesPasswordExt, ".pwd")
	ctx := context.Background()

	listener := make(chan ethtypes.Address0xHex, 1)
	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig), listener)
	assert.NoError(t, err)
	if init {
		err = ff.Initialize(ctx)
		assert.NoError(t, err)
	}

	return ctx, ff.(*fsWallet), listener, func() {
		ff.Close()
	}
}

func TestFileListener(t *testing.T) {

	ctx, f, listener1, done := newEmptyWalletTestDir(t, true)
	defer done()

	// add a 2nd listener
	listener2 := make(chan ethtypes.Address0xHex, 1)
	f.AddListener(listener2)

	testPWFIle, err := ioutil.ReadFile("../../test/keystore_toml/1f185718734552d08278aa70f804580bab5fd2b4.pwd")
	assert.NoError(t, err)

	err = ioutil.WriteFile(path.Join(f.conf.Path, "1f185718734552d08278aa70f804580bab5fd2b4.pwd"), testPWFIle, 0644)
	assert.NoError(t, err)

	testKeyFIle, err := ioutil.ReadFile("../../test/keystore_toml/1f185718734552d08278aa70f804580bab5fd2b4.key.json")
	assert.NoError(t, err)

	err = ioutil.WriteFile(path.Join(f.conf.Path, "1f185718734552d08278aa70f804580bab5fd2b4.key.json"), testKeyFIle, 0644)
	assert.NoError(t, err)

	newAddr1 := <-listener1
	assert.Equal(t, `0x1f185718734552d08278aa70f804580bab5fd2b4`, newAddr1.String())
	newAddr2 := <-listener2
	assert.Equal(t, `0x1f185718734552d08278aa70f804580bab5fd2b4`, newAddr2.String())

	addr := *ethtypes.MustNewAddress(`1f185718734552d08278aa70f804580bab5fd2b4`)
	wf, err := f.GetWalletFile(ctx, addr)
	assert.NoError(t, err)
	assert.Equal(t, wf.KeyPair().Address, addr)

}

func TestFileListenerStartFail(t *testing.T) {

	ctx, f, _, done := newEmptyWalletTestDir(t, false)
	defer done()

	os.RemoveAll(f.conf.Path)
	err := f.Initialize(ctx)
	assert.Regexp(t, "FF22060", err)

}

func TestFileListenerRemoveDirWhileListening(t *testing.T) {

	ctx, f, _, done := newEmptyWalletTestDir(t, true)
	defer done()

	errs := make(chan error, 1)
	errs <- fmt.Errorf("pop")
	ctx, cancelCtx := context.WithCancel(ctx)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancelCtx()
	}()
	f.fsListenerLoop(ctx, func() {}, make(chan fsnotify.Event), errs)

}
