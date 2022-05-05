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

package filewallet

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

func newTestFilenameOnlyWallet(t *testing.T) (context.Context, *fileWallet, func()) {
	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".key.json")
	config.Set(signerconfig.FileWalletFilenamesPasswordExt, ".pwd")
	ctx := context.Background()

	ff, err := NewFileWallet(ctx)
	assert.NoError(t, err)
	return ctx, ff.(*fileWallet), func() {
		ff.Close()
	}
}

func newTestTOMLMetadataWallet(t *testing.T) (context.Context, *fileWallet, func()) {
	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".toml")
	config.Set(signerconfig.FileWalletMetadataKeyFileProperty, `{{ index .signing "key-file" }}`)
	config.Set(signerconfig.FileWalletMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)
	ctx := context.Background()

	ff, err := NewFileWallet(ctx)
	assert.NoError(t, err)
	return ctx, ff.(*fileWallet), func() {
		ff.Close()
	}
}

func TestGetAccountSimpleFilenamesOK(t *testing.T) {

	ctx, f, done := newTestFilenameOnlyWallet(t)
	defer done()

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestListAccountsTOMLOk(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
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

}

func TestListAccountsBadDir(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.path = "!!!"
	_, err := f.GetAccounts(ctx)
	assert.Regexp(t, "FF22013", err)

}

func TestRefreshOK(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	err := f.Refresh(ctx)
	assert.NoError(t, err)

}

func TestSignOK(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()

	err := f.Initialize(ctx)
	assert.NoError(t, err)

	b, err := f.Sign(ctx, &ethsigner.Transaction{
		From: ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"),
	}, 2022)
	assert.NoError(t, err)
	assert.NotNil(t, b)

}

func TestSignNotFound(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()

	_, err := f.Sign(ctx, &ethsigner.Transaction{
		From: ethtypes.MustNewAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}, 2022)
	assert.Regexp(t, "FF22014", err)

}

func TestGetAccountCached(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()

	s, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// 2nd time is cached
	s, err = f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.NoError(t, err)
	assert.NotNil(t, s)

}

func TestGetAccountBadYAML(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataFormat = "yaml"

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountBadJSON(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataFormat = "json"

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestInitBadKeyFileTemplate(t *testing.T) {

	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".toml")
	config.Set(signerconfig.FileWalletMetadataKeyFileProperty, `{{ !!! }}`)
	config.Set(signerconfig.FileWalletMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)

	ctx := context.Background()
	_, err := NewFileWallet(ctx)
	assert.Regexp(t, "FF22016.*keyFileProperty", err)
}

func TestInitBadPasswordFileTemplate(t *testing.T) {

	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".toml")
	config.Set(signerconfig.FileWalletMetadataKeyFileProperty, `{{ index .signing "key-file" }}`)
	config.Set(signerconfig.FileWalletMetadataPasswordFileProperty, `{{ !!! }}`)

	ctx := context.Background()
	_, err := NewFileWallet(ctx)
	assert.Regexp(t, "FF22016.*passwordFileProperty", err)
}

func TestGetAccountBadTOMLRefKey(t *testing.T) {

	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".toml")
	config.Set(signerconfig.FileWalletMetadataKeyFileProperty, `{{ index .signing "wrong" }}`)
	config.Set(signerconfig.FileWalletMetadataPasswordFileProperty, `{{ index .signing "password-file" }}`)

	ctx := context.Background()
	ff, err := NewFileWallet(ctx)
	defer ff.Close()
	assert.NoError(t, err)
	f := ff.(*fileWallet)

	_, err = f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)
}

func TestGetAccountNoTemplates(t *testing.T) {

	signerconfig.Reset()
	config.Set(signerconfig.FileWalletPath, "../../test/keystore_toml")
	config.Set(signerconfig.FileWalletFilenamesPrimaryExt, ".toml")

	ctx := context.Background()
	ff, err := NewFileWallet(ctx)
	defer ff.Close()
	assert.NoError(t, err)
	f := ff.(*fileWallet)

	_, err = f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)
}

func TestGetAccountBadKeyfile(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataFormat = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.defaultPasswordFile = "../../test/keystore_toml/1f185718734552d08278aa70f804580bab5fd2b4.pwd"

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountBadDefaultPasswordfile(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataFormat = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.defaultPasswordFile = "!!!"

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountNoPassword(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataPasswordFileProperty = nil

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0x1f185718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountWrongPath(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataPasswordFileProperty = nil

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("5d093e9b41911be5f5c4cf91b108bac5d130fa83"))
	assert.Regexp(t, "FF22015", err)

}

func TestGetAccountNotFound(t *testing.T) {

	ctx, f, done := newTestTOMLMetadataWallet(t)
	defer done()
	f.metadataFormat = "none" // tell it to read the TOML directly as a kv3
	f.metadataPasswordFileProperty = nil
	f.defaultPasswordFile = "!!!"

	_, err := f.getSignerForAccount(ctx, ethtypes.MustNewAddress("0xFFFF5718734552d08278aa70f804580bab5fd2b4"))
	assert.Regexp(t, "FF22014", err)

}
