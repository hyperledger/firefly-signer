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
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newTestRegexpFilenameOnlyWallet(t *testing.T, init bool) (context.Context, *fsWallet, func()) {
	config.RootConfigReset()
	logrus.SetLevel(logrus.TraceLevel)

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryMatchRegex, "^((0x)?[0-9a-z]+).key.json$")
	unitTestConfig.Set(ConfigFilenamesPasswordExt, ".pwd")
	unitTestConfig.Set(ConfigDisableListener, true)
	ctx := context.Background()

	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	assert.NoError(t, err)
	if init {
		err = ff.Initialize(ctx)
		assert.NoError(t, err)
	}

	return ctx, ff.(*fsWallet), func() {
		ff.Close()
	}
}

func newTestTOMLMetadataWallet(t *testing.T, init bool) (context.Context, *fsWallet, func()) {
	config.RootConfigReset()
	logrus.SetLevel(logrus.TraceLevel)

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")
	unitTestConfig.Set(ConfigMetadataKeyFileProperty, `{{ index .signing "key-file" }}`)
	unitTestConfig.Set(ConfigMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)
	unitTestConfig.Set(ConfigDisableListener, true)
	ctx := context.Background()

	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	assert.NoError(t, err)
	if init {
		err = ff.Initialize(ctx)
		assert.NoError(t, err)
	}
	return ctx, ff.(*fsWallet), func() {
		ff.Close()
	}
}

func TestGetAccountSimpleFilenamesOK(t *testing.T) {

	ctx, f, done := newTestRegexpFilenameOnlyWallet(t, true)
	defer done()

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.NoError(t, err)

}

func TestGetAccountSimpleFilenamesMissingPWD(t *testing.T) {

	ctx, f, done := newTestRegexpFilenameOnlyWallet(t, true)
	defer done()

	f.conf.Filenames.PasswordExt = ".wrong"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountSimpleFilenamesMismatchAddress(t *testing.T) {

	ctx, f, done := newTestRegexpFilenameOnlyWallet(t, true)
	defer done()

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"abcd1234abcd1234abcd1234abcd1234abcd1234"`))
	assert.Regexp(t, "FF22059", err)

}

func TestListAccountsTOMLOk(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	accounts, err := f.GetAccounts(ctx)
	assert.NoError(t, err)
	assert.Len(t, accounts, 3)
	all := map[string]bool{}
	for _, a := range accounts {
		all[a.String()] = true
	}
	assert.True(t, all["0x1f185718734552d08278aa70f804580bab5fd2b4"])
	assert.True(t, all["0x497eedc4299dea2f2a364be10025d0ad0f702de3"])
	assert.True(t, all["0x5d093e9b41911be5f5c4cf91b108bac5d130fa83"])

	_, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.NoError(t, err)
	_, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x497eedc4299dea2f2a364be10025d0ad0f702de3"`))
	assert.Regexp(t, "FF22015", err)
	_, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x5d093e9b41911be5f5c4cf91b108bac5d130fa83"`))
	assert.Regexp(t, "FF22015", err)

}

func TestBadRegexp(t *testing.T) {

	_, err := NewFilesystemWallet(context.Background(), &Config{
		Path: "../../test/keystore_toml",
		Filenames: FilenamesConfig{
			PrimaryMatchRegex: "[[[[!bad",
		},
	})
	assert.Regexp(t, "FF22056", err)

}

func TestMissingCaptureRegexp(t *testing.T) {

	_, err := NewFilesystemWallet(context.Background(), &Config{
		Path: "../../test/keystore_toml",
		Filenames: FilenamesConfig{
			PrimaryMatchRegex: ".*",
		},
	})
	assert.Regexp(t, "FF22057", err)

}

func TestListAccountsBadDir(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, false)
	defer done()
	f.conf.Path = "!!!"
	err := f.Initialize(ctx)
	assert.Regexp(t, "FF22013", err)

}

func TestRefreshOK(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	err := f.Refresh(ctx)
	assert.NoError(t, err)

}

func TestRefreshStatFail(t *testing.T) {

	config.RootConfigReset()
	logrus.SetLevel(logrus.TraceLevel)

	tmpDir := t.TempDir()
	os.Mkdir(path.Join(tmpDir, "baddir"), 0000)

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, tmpDir)
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")
	unitTestConfig.Set(ConfigMetadataKeyFileProperty, `{{ index .signing "key-file" }}`)
	unitTestConfig.Set(ConfigMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)
	unitTestConfig.Set(ConfigDisableListener, true)
	ctx := context.Background()

	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	assert.NoError(t, err)
	defer ff.Close()

	err = ff.Refresh(ctx)
	assert.NoError(t, err)

}

func TestSignOK(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()

	b, err := f.Sign(ctx, &ethsigner.Transaction{
		From: json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`),
	}, 2022)
	assert.NoError(t, err)
	assert.NotNil(t, b)

}

func TestSignNotFound(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()

	_, err := f.Sign(ctx, &ethsigner.Transaction{
		From: json.RawMessage(`"0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"`),
	}, 2022)
	assert.Regexp(t, "FF22014", err)

}

func TestGetAccountCached(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()

	s, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// 2nd time is cached
	s, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.NoError(t, err)
	assert.NotNil(t, s)

}

func TestGetAccountBadYAML(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.conf.Metadata.Format = "yaml"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountBadAddress(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"bad address"`))
	assert.Regexp(t, "bad address", err)

}

func TestGetAccountBadJSON(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.conf.Metadata.Format = "json"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestInitBadKeyFileTemplate(t *testing.T) {

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")
	unitTestConfig.Set(ConfigMetadataKeyFileProperty, `{{ !!! }}`)
	unitTestConfig.Set(ConfigMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)

	ctx := context.Background()
	_, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	assert.Regexp(t, "FF22016.*keyFileProperty", err)
}

func TestInitBadPasswordFileTemplate(t *testing.T) {

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")
	unitTestConfig.Set(ConfigMetadataKeyFileProperty, `{{ index .signing "key-file" }}`)
	unitTestConfig.Set(ConfigMetadataPasswordFileProperty, `{{ !!! }}`)

	ctx := context.Background()
	_, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	assert.Regexp(t, "FF22016.*passwordFileProperty", err)
}

func TestGetAccountBadTOMLRefKey(t *testing.T) {

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")
	unitTestConfig.Set(ConfigMetadataKeyFileProperty, `{{ index .signing "wrong" }}`)
	unitTestConfig.Set(ConfigMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)

	ctx := context.Background()
	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	defer ff.Close()
	assert.NoError(t, err)
	f := ff.(*fsWallet)

	_, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22014", err)
}

func TestGetAccountNoTemplates(t *testing.T) {

	unitTestConfig := config.RootSection("ut_fs_config")
	InitConfig(unitTestConfig)
	unitTestConfig.Set(ConfigPath, "../../test/keystore_toml")
	unitTestConfig.Set(ConfigFilenamesPrimaryExt, ".toml")

	ctx := context.Background()
	ff, err := NewFilesystemWallet(ctx, ReadConfig(unitTestConfig))
	defer ff.Close()
	assert.NoError(t, err)
	f := ff.(*fsWallet)

	_, err = f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22014", err)
}

func TestGetAccountBadKeyfile(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.conf.Metadata.Format = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.conf.DefaultPasswordFile = "../../test/keystore_toml/1f185718734552d08278aa70f804580bab5fd2b4.pwd"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountBadDefaultPasswordfile(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.conf.Metadata.Format = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.conf.DefaultPasswordFile = "!!!"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountNoPassword(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.metadataPasswordFileProperty = nil

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0x1f185718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountWrongPath(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.metadataPasswordFileProperty = nil

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"5d093e9b41911be5f5c4cf91b108bac5d130fa83"`))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountNotFound(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t, true)
	defer done()
	f.conf.Metadata.Format = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.conf.DefaultPasswordFile = "!!!"

	_, err := f.getSignerForAccount(ctx, json.RawMessage(`"0xFFFF5718734552d08278aa70f804580bab5fd2b4"`))
	assert.Regexp(t, "FF22014", err)

}

func TestLoadKeyBadPath(t *testing.T) {

	ctx, f, done := newTestRegexpFilenameOnlyWallet(t, false)
	defer done()

	_, err := f.loadWalletFile(ctx, *ethtypes.MustNewAddress("0xFFFF5718734552d08278aa70f804580bab5fd2b4"), "../../test/keystore_toml/wrong.txt")
	assert.Regexp(t, "FF22015", err)

}
