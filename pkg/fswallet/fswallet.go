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

package fswallet

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/eip712"
	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/keystorev3"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/karlseguin/ccache"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

type SyncAddressCallback func(context.Context, ethtypes.Address0xHex) error

// Wallet is a directory containing a set of KeystoreV3 files, conforming
// to the ethsigner.Wallet interface and providing notifications when new
// keys are added to the wallet (via FS listener).
type Wallet interface {
	ethsigner.WalletTypedData
	GetWalletFile(ctx context.Context, addr ethtypes.Address0xHex) (keystorev3.WalletFile, error)
	SetSyncAddressCallback(SyncAddressCallback)
	AddListener(listener chan<- ethtypes.Address0xHex)
}

func NewFilesystemWallet(ctx context.Context, conf *Config, initialListeners ...chan<- ethtypes.Address0xHex) (ww Wallet, err error) {
	w := &fsWallet{
		conf:             *conf,
		listeners:        initialListeners,
		addressToFileMap: make(map[ethtypes.Address0xHex]string),
	}
	w.signerCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(fftypes.ParseToByteSize(conf.SignerCacheSize)),
	)
	w.metadataKeyFileProperty, err = goTemplateFromConfig(ctx, ConfigMetadataKeyFileProperty, conf.Metadata.KeyFileProperty)
	if err != nil {
		return nil, err
	}
	w.metadataPasswordFileProperty, err = goTemplateFromConfig(ctx, ConfigMetadataPasswordFileProperty, conf.Metadata.PasswordFileProperty)
	if err != nil {
		return nil, err
	}
	if conf.Filenames.PrimaryMatchRegex != "" {
		if w.primaryMatchRegex, err = regexp.Compile(conf.Filenames.PrimaryMatchRegex); err != nil {
			return nil, i18n.NewError(ctx, signermsgs.MsgBadRegularExpression, ConfigFilenamesPrimaryMatchRegex, err)
		}
		if len(w.primaryMatchRegex.SubexpNames()) < 2 {
			return nil, i18n.NewError(ctx, signermsgs.MsgMissingRegexpCaptureGroup, w.primaryMatchRegex.String())
		}
	}
	return w, nil
}

func goTemplateFromConfig(ctx context.Context, name string, templateStr string) (*template.Template, error) {
	if templateStr == "" {
		return nil, nil
	}
	t, err := template.New(name).Parse(templateStr)
	if err != nil {
		return nil, i18n.NewError(ctx, signermsgs.MsgBadGoTemplate, name)
	}
	return t, nil
}

type fsWallet struct {
	conf                         Config
	signerCache                  *ccache.Cache
	signerCacheTTL               time.Duration
	metadataKeyFileProperty      *template.Template
	metadataPasswordFileProperty *template.Template
	primaryMatchRegex            *regexp.Regexp
	syncAddressCallback          SyncAddressCallback

	mux               sync.Mutex
	addressToFileMap  map[ethtypes.Address0xHex]string // map for lookup to filename
	addressList       []*ethtypes.Address0xHex         // ordered list in filename at startup, then notification order
	listeners         []chan<- ethtypes.Address0xHex
	fsListenerCancel  context.CancelFunc
	fsListenerStarted chan error
	fsListenerDone    chan struct{}
}

func (w *fsWallet) Sign(ctx context.Context, txn *ethsigner.Transaction, chainID int64) ([]byte, error) {
	keypair, err := w.getSignerForJSONAccount(ctx, txn.From)
	if err != nil {
		return nil, err
	}
	return txn.Sign(keypair, chainID)
}

func (w *fsWallet) SignTypedDataV4(ctx context.Context, from ethtypes.Address0xHex, payload *eip712.TypedData) (*ethsigner.EIP712Result, error) {
	keypair, err := w.getSignerForAddr(ctx, from)
	if err != nil {
		return nil, err
	}
	return ethsigner.SignTypedDataV4(ctx, keypair, payload)
}

func (w *fsWallet) Initialize(ctx context.Context) error {
	// Run a get accounts pass, to check all is ok
	lCtx, lCancel := context.WithCancel(log.WithLogField(ctx, "fswallet", w.conf.Path))
	w.fsListenerCancel = lCancel
	w.fsListenerStarted = make(chan error)
	w.fsListenerDone = make(chan struct{})
	// Make sure listener is listening for changes, before doing the scan
	if err := w.startFilesystemListener(lCtx); err != nil {
		return err
	}
	// Do an initial full scan before returning
	return w.Refresh(ctx)
}

// Asynchronously listen for all addresses as they are detected - during startup, or after startup
func (w *fsWallet) AddListener(listener chan<- ethtypes.Address0xHex) {
	w.mux.Lock()
	defer w.mux.Unlock()
	w.listeners = append(w.listeners, listener)
}

// As an alternative to registering a listener are able to supply a single *synchronous* callback
// that will process all of the addresses that exist on-disk at the time of refresh in-line.
// This is very useful if you want to be sure your application using this module does not advertise it
// is available until it has built a lookup map for all of the files that existed before it started.
//
// This will (by definition) delay initialize/refresh while that processing happens.
//
// This function is called under the lock of addresses, so your processing should be efficient.
func (w *fsWallet) SetSyncAddressCallback(callback SyncAddressCallback) {
	w.syncAddressCallback = callback
}

// GetAccounts returns the currently cached list of known addresses
func (w *fsWallet) GetAccounts(_ context.Context) ([]*ethtypes.Address0xHex, error) {
	w.mux.Lock()
	defer w.mux.Unlock()
	accounts := make([]*ethtypes.Address0xHex, len(w.addressList))
	copy(accounts, w.addressList)
	return accounts, nil
}

func (w *fsWallet) matchFilename(ctx context.Context, f fs.FileInfo) *ethtypes.Address0xHex {
	if f.IsDir() {
		log.L(ctx).Tracef("Ignoring '%s/%s: directory", w.conf.Path, f.Name())
		return nil
	}
	if w.primaryMatchRegex != nil {
		match := w.primaryMatchRegex.FindStringSubmatch(f.Name())
		if match == nil {
			log.L(ctx).Tracef("Ignoring '%s/%s': does not match regexp", w.conf.Path, f.Name())
			return nil
		}
		addr, err := ethtypes.NewAddress(match[1]) // safe due to SubexpNames() length check
		if err != nil {
			log.L(ctx).Warnf("Ignoring '%s/%s': invalid address '%s': %s", w.conf.Path, f.Name(), match[1], err)
			return nil
		}
		return addr
	}
	if !strings.HasSuffix(f.Name(), w.conf.Filenames.PrimaryExt) {
		log.L(ctx).Tracef("Ignoring '%s/%s: does not match extension '%s'", w.conf.Path, f.Name(), w.conf.Filenames.PrimaryExt)
	}
	addrString := strings.TrimSuffix(f.Name(), w.conf.Filenames.PrimaryExt)
	addr, err := ethtypes.NewAddress(addrString)
	if err != nil {
		log.L(ctx).Warnf("Ignoring '%s/%s': invalid address '%s': %s", w.conf.Path, f.Name(), addrString, err)
		return nil
	}
	return addr
}

func (w *fsWallet) Refresh(ctx context.Context) error {
	log.L(ctx).Infof("Refreshing account list at %s", w.conf.Path)
	dirEntries, err := os.ReadDir(w.conf.Path)
	if err != nil {
		return i18n.WrapError(ctx, err, signermsgs.MsgReadDirFile)
	}
	files := make([]os.FileInfo, 0, len(dirEntries))
	for _, de := range dirEntries {
		fi, infoErr := de.Info()
		if infoErr == nil {
			files = append(files, fi)
		}
	}
	return w.notifyNewFiles(ctx, files...)
}

func (w *fsWallet) notifyNewFiles(ctx context.Context, files ...fs.FileInfo) error {
	if len(files) == 0 {
		return nil
	}

	// Lock now we have the list
	w.mux.Lock()
	defer w.mux.Unlock()
	newAddresses := make([]*ethtypes.Address0xHex, 0)
	for _, f := range files {
		addr := w.matchFilename(ctx, f)
		if addr != nil {
			if existingFilename, exists := w.addressToFileMap[*addr]; existingFilename != f.Name() {
				w.addressToFileMap[*addr] = f.Name()
				if !exists {
					log.L(ctx).Debugf("Added address: %s (file=%s)", addr, f.Name())
					w.addressList = append(w.addressList, addr)
					newAddresses = append(newAddresses, addr)
				}
			}
		}
	}
	listeners := make([]chan<- ethtypes.Address0xHex, len(w.listeners))
	copy(listeners, w.listeners)
	log.L(ctx).Debugf("Processed %d files. Found %d new addresses", len(files), len(newAddresses))

	// Avoid holding the lock while calling the async listeners, by using a go-routine
	go func() {
		for _, l := range w.listeners {
			for _, addr := range newAddresses {
				l <- *addr
			}
		}
	}()

	// Sync callbacks are called here in-line, with the lock.
	if w.syncAddressCallback != nil {
		for _, addr := range newAddresses {
			if err := w.syncAddressCallback(ctx, *addr); err != nil {
				log.L(ctx).Errorf("sync listener returned error for address %s: %s", addr, err)
				return err
			}
		}
	}

	return nil
}

func (w *fsWallet) Close() error {
	if w.fsListenerCancel != nil {
		w.fsListenerCancel()
		<-w.fsListenerDone
	}
	return nil
}

func (w *fsWallet) getSignerForJSONAccount(ctx context.Context, rawAddrJSON json.RawMessage) (*secp256k1.KeyPair, error) {

	// We require an ethereum address in the "from" field
	var from ethtypes.Address0xHex
	err := json.Unmarshal(rawAddrJSON, &from)
	if err != nil {
		return nil, err
	}
	return w.getSignerForAddr(ctx, from)
}

func (w *fsWallet) getSignerForAddr(ctx context.Context, from ethtypes.Address0xHex) (*secp256k1.KeyPair, error) {

	wf, err := w.GetWalletFile(ctx, from)
	if err != nil {
		return nil, err
	}
	return wf.KeyPair(), nil

}

func (w *fsWallet) GetWalletFile(ctx context.Context, addr ethtypes.Address0xHex) (keystorev3.WalletFile, error) {

	addrString := addr.String()
	cached := w.signerCache.Get(addrString)
	if cached != nil {
		cached.Extend(w.signerCacheTTL)
		return cached.Value().(keystorev3.WalletFile), nil
	}

	w.mux.Lock()
	primaryFilename, ok := w.addressToFileMap[addr]
	w.mux.Unlock()
	if !ok {
		return nil, i18n.NewError(ctx, signermsgs.MsgWalletNotAvailable, addr)
	}

	kv3, err := w.loadWalletFile(ctx, addr, path.Join(w.conf.Path, primaryFilename))
	if err != nil {
		return nil, err
	}

	keypair := kv3.KeyPair()
	if keypair.Address != addr {
		return nil, i18n.NewError(ctx, signermsgs.MsgAddressMismatch, keypair.Address, addr)
	}

	w.signerCache.Set(addrString, kv3, w.signerCacheTTL)
	return kv3, err

}

func (w *fsWallet) loadWalletFile(ctx context.Context, addr ethtypes.Address0xHex, primaryFilename string) (keystorev3.WalletFile, error) {

	b, err := os.ReadFile(primaryFilename)
	if err != nil {
		log.L(ctx).Errorf("Failed to read '%s': %s", primaryFilename, err)
		return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
	}

	keyFilename, passwordFilename, err := w.getKeyAndPasswordFiles(ctx, addr, primaryFilename, b)
	if err != nil {
		return nil, err
	}
	log.L(ctx).Debugf("Reading keyfile=%s passwordfile=%s", keyFilename, passwordFilename)

	if keyFilename != primaryFilename {
		b, err = os.ReadFile(keyFilename)
		if err != nil {
			log.L(ctx).Errorf("Failed to read '%s' (keyfile): %s", keyFilename, err)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
		}
	}

	var password []byte
	if passwordFilename != "" {
		password, err = os.ReadFile(passwordFilename)
		if err != nil {
			log.L(ctx).Debugf("Failed to read '%s' (password file): %s", passwordFilename, err)
		} else if w.conf.Filenames.PasswordTrimSpace {
			password = []byte(strings.TrimSpace(string(password)))
		}
	}

	// fall back to default password file
	if password == nil {
		if w.conf.DefaultPasswordFile == "" {
			log.L(ctx).Errorf("No password file available for address, and no default password file: %s", addr)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
		}
		password, err = os.ReadFile(w.conf.DefaultPasswordFile)
		if err != nil {
			log.L(ctx).Errorf("Failed to read '%s' (default password file): %s", w.conf.DefaultPasswordFile, err)
			return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
		}

	}

	// Ok - now we have what we need to open up the keyfile
	kv3, err := keystorev3.ReadWalletFile(b, password)
	if err != nil {
		log.L(ctx).Errorf("Failed to read '%s' (bad keystorev3 file): %s", w.conf.DefaultPasswordFile, err)
		return nil, i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
	}
	log.L(ctx).Infof("Loaded signing key for address: %s", addr)
	return kv3, nil

}

func (w *fsWallet) getKeyAndPasswordFiles(ctx context.Context, addr ethtypes.Address0xHex, primaryFilename string, primaryFile []byte) (kf string, pf string, err error) {
	if strings.ToLower(w.conf.Metadata.Format) == "auto" {
		w.conf.Metadata.Format = strings.TrimPrefix(w.conf.Filenames.PrimaryExt, ".")
	}

	var metadata map[string]interface{}
	switch w.conf.Metadata.Format {
	case "toml", "tml":
		err = toml.Unmarshal(primaryFile, &metadata)
	case "json":
		err = json.Unmarshal(primaryFile, &metadata)
	case "yaml", "yml":
		err = yaml.Unmarshal(primaryFile, &metadata)
	default:
		// No separate metadata file - we just use the default password file extension instead
		passwordPath := w.conf.Filenames.PasswordPath
		if passwordPath == "" {
			passwordPath = w.conf.Path
		}
		passwordFilename := addr.String()
		if !w.conf.Filenames.With0xPrefix {
			passwordFilename = strings.TrimPrefix(passwordFilename, "0x")
		}
		passwordFilename += w.conf.Filenames.PasswordExt
		return primaryFilename, path.Join(passwordPath, passwordFilename), nil
	}
	if err != nil {
		log.L(ctx).Errorf("Failed to parse '%s' as %s: %s", primaryFilename, w.conf.Metadata.Format, err)
		return "", "", i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
	}

	kf, err = w.goTemplateToString(ctx, primaryFilename, metadata, w.metadataKeyFileProperty)
	if err == nil {
		pf, err = w.goTemplateToString(ctx, primaryFilename, metadata, w.metadataPasswordFileProperty)
	}
	if err != nil || kf == "" {
		return "", "", i18n.NewError(ctx, signermsgs.MsgWalletFailed, addr)
	}
	return kf, pf, nil
}

func (w *fsWallet) goTemplateToString(ctx context.Context, filename string, data map[string]interface{}, t *template.Template) (string, error) {
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
