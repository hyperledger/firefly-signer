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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/keystorev3"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/karlseguin/ccache"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

func NewFileWallet(ctx context.Context) (ww ethsigner.Wallet, err error) {
	w := &fileWallet{
		ctx:            ctx,
		signerCacheTTL: config.GetDuration(signerconfig.FileWalletSignerCacheTTL),

		path:                config.GetString(signerconfig.FileWalletPath),
		filenames0xPrefix:   config.GetBool(signerconfig.FileWalletFilenamesWith0xPrefix),
		extensionPrimary:    config.GetString(signerconfig.FileWalletFilenamesPrimaryExt),
		extensionPassword:   config.GetString(signerconfig.FileWalletFilenamesPasswordExt),
		defaultPasswordFile: config.GetString(signerconfig.FileWalletDefaultPasswordFile),
		metadataFormat:      config.GetString(signerconfig.FileWalletMetadataFormat),
	}
	w.signerCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(signerconfig.FileWalletSignerCacheSize)),
	)
	w.metadataKeyFileProperty, err = goTemplateFromConfig(ctx, signerconfig.FileWalletMetadataKeyFileProperty)
	if err != nil {
		return nil, err
	}
	w.metadataPasswordFileProperty, err = goTemplateFromConfig(ctx, signerconfig.FileWalletMetadataPasswordFileProperty)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func goTemplateFromConfig(ctx context.Context, configKey config.RootKey) (*template.Template, error) {
	templateStr := config.GetString(configKey)
	if templateStr == "" {
		return nil, nil
	}
	t, err := template.New(string(configKey)).Parse(templateStr)
	if err != nil {
		return nil, i18n.NewError(ctx, signermsgs.MsgBadGoTemplate, configKey)
	}
	return t, nil
}

type fileWallet struct {
	ctx            context.Context
	signerCache    *ccache.Cache
	signerCacheTTL time.Duration

	path                         string
	filenames0xPrefix            bool
	extensionPrimary             string
	extensionPassword            string
	defaultPasswordFile          string
	metadataFormat               string
	metadataKeyFileProperty      *template.Template
	metadataPasswordFileProperty *template.Template
}

func (w *fileWallet) Sign(ctx context.Context, txn *ethsigner.Transaction, chainID int64) ([]byte, error) {
	keypair, err := w.getSignerForAccount(ctx, txn.From)
	if err != nil {
		return nil, err
	}
	return txn.Sign(keypair, chainID)
}

func (w *fileWallet) Initialize(ctx context.Context) error {
	// Run a get accounts pass, to check all is ok
	return w.Refresh(ctx)
}

func (w *fileWallet) GetAccounts(ctx context.Context) ([]*ethtypes.Address0xHex, error) {
	addresses := []*ethtypes.Address0xHex{}
	files, err := ioutil.ReadDir(w.path)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, signermsgs.MsgReadDirFile)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), w.extensionPrimary) {
			addrString := strings.TrimSuffix(f.Name(), w.extensionPrimary)
			addr, err := ethtypes.NewAddress(addrString)
			if err != nil {
				log.L(ctx).Warnf("Invalid filename in wallet directory: %s/%s", w.path, f.Name())
				continue
			}
			addresses = append(addresses, addr)
		}
	}
	return addresses, nil
}

func (w *fileWallet) Refresh(ctx context.Context) error {
	_, err := w.GetAccounts(ctx)
	return err
}

func (w *fileWallet) Close() error {
	return nil
}

func (w *fileWallet) getSignerForAccount(ctx context.Context, addr *ethtypes.Address0xHex) (*secp256k1.KeyPair, error) {

	addrString := addr.String()
	if !w.filenames0xPrefix {
		addrString = strings.TrimPrefix(addrString, "0x")
	}

	cached := w.signerCache.Get(addrString)
	if cached != nil {
		cached.Extend(w.signerCacheTTL)
		return cached.Value().(*secp256k1.KeyPair), nil
	}

	keypair, err := w.lookupKeyByAddress(ctx, addrString)
	if err != nil {
		return nil, err
	}
	w.signerCache.Set(addrString, keypair, w.signerCacheTTL)
	return keypair, err

}

func (w *fileWallet) lookupKeyByAddress(ctx context.Context, addrString string) (*secp256k1.KeyPair, error) {

	primaryFilename := fmt.Sprintf("%s%s", addrString, w.extensionPrimary)
	b, err := ioutil.ReadFile(path.Join(w.path, primaryFilename))
	if err != nil {
		log.L(ctx).Errorf("Failed to read '%s' (key or metadata): %s", primaryFilename, err)
		return nil, i18n.NewError(ctx, signermsgs.MsgWalletNotAvailable, addrString)
	}

	keyFilename, passwordFilename, err := w.getKeyAndPasswordFiles(ctx, addrString, primaryFilename, b)
	if err != nil {
		return nil, err
	}
	log.L(ctx).Debugf("Reading keyfile=%s passwordfile=%s", keyFilename, passwordFilename)

	if keyFilename != primaryFilename {
		b, err = ioutil.ReadFile(keyFilename)
		if err != nil {
			log.L(ctx).Errorf("Failed to read '%s' (keyfile): %s", keyFilename, err)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
		}
	}

	var password []byte
	if passwordFilename != "" {
		password, err = ioutil.ReadFile(passwordFilename)
		if err != nil {
			log.L(ctx).Debugf("Failed to read '%s' (password file): %s", passwordFilename, err)
		}
	}

	// fall back to default password file
	if password == nil {
		if w.defaultPasswordFile == "" {
			log.L(ctx).Errorf("No password file available for address, and no default password file: %s", addrString)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
		}
		password, err = ioutil.ReadFile(w.defaultPasswordFile)
		if err != nil {
			log.L(ctx).Errorf("Failed to read '%s' (default password file): %s", w.defaultPasswordFile, err)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
		}

	}

	// Ok - now we have what we need to open up the keyfile
	kv3, err := keystorev3.ReadWalletFile(b, password)
	if err != nil {
		log.L(ctx).Errorf("Failed to read '%s' (bad keystorev3 file): %s", w.defaultPasswordFile, err)
		return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
	}
	return kv3.KeyPair(), nil

}

func (w *fileWallet) getKeyAndPasswordFiles(ctx context.Context, addrString string, primaryFilename string, primaryFile []byte) (kf string, pf string, err error) {
	if strings.ToLower(w.metadataFormat) == "auto" {
		w.metadataFormat = strings.TrimPrefix(w.extensionPrimary, ".")
	}

	var metadata map[string]interface{}
	switch w.metadataFormat {
	case "toml", "tml":
		err = toml.Unmarshal(primaryFile, &metadata)
	case "json":
		err = json.Unmarshal(primaryFile, &metadata)
	case "yaml", "yml":
		err = yaml.Unmarshal(primaryFile, &metadata)
	default:
		// No separate metadata file - we just use the default password file extension instead
		passwordFilename := ""
		if w.extensionPassword != "" {
			passwordFilename = fmt.Sprintf("%s%s", addrString, w.extensionPassword)
		}
		return primaryFilename, passwordFilename, nil
	}
	if err != nil {
		log.L(ctx).Errorf("Failed to parse '%s' as %s: %s", primaryFilename, w.metadataFormat, err)
		return "", "", i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
	}

	kf, err = w.goTemplateToString(ctx, primaryFilename, metadata, w.metadataKeyFileProperty)
	if err == nil {
		pf, err = w.goTemplateToString(ctx, primaryFilename, metadata, w.metadataPasswordFileProperty)
	}
	if err != nil || kf == "" {
		return "", "", i18n.NewError(ctx, signermsgs.MsgWalletFailed, addrString)
	}
	return kf, pf, nil
}

func (w *fileWallet) goTemplateToString(ctx context.Context, filename string, data map[string]interface{}, t *template.Template) (string, error) {
	if t == nil {
		return "", nil
	}
	buff := new(strings.Builder)
	err := t.Execute(buff, data)
	val := buff.String()
	if strings.Contains(val, "<no value>") || err != nil {
		log.L(ctx).Errorf("Failed to execute go template against metadata file %s: err=%v", filename, err)
		return "", nil
	}
	return val, err
}
