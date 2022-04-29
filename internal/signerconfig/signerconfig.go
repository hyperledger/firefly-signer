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

package signerconfig

import (
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/httpserver"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/spf13/viper"
)

var ffc = config.AddRootKey

var (
	// FileWalletEnabled if the Keystore V3 wallet is enabled
	FileWalletEnabled = ffc("filewallet.enabled")
	// FileWalletPath the path of the Keystore V3 wallet path
	FileWalletPath = ffc("filewallet.path")
	// FileWalletFilenamesWith0xPrefix extension to append to the "from" address string to find the file (see metadata section for file types). All filenames must be lower case on disk.
	FileWalletFilenamesWith0xPrefix = ffc("filewallet.filenames.with0xPrefix")
	// FileWalletFilenamesPrimaryExt extension to append to the "from" address string to find the file (see metadata section for file types). All filenames must be lower case on disk.
	FileWalletFilenamesPrimaryExt = ffc("filewallet.filenames.primaryExt")
	// FileWalletFilenamesPasswordExt extension to append to the "from" address string to find the password file (if not using a metadata file to specify the password file)
	FileWalletFilenamesPasswordExt = ffc("filewallet.filenames.passwordExt")
	// FileWalletDefaultPasswordFile default password file to use if neither the metadata, or passwordExtension find a password
	FileWalletDefaultPasswordFile = ffc("filewallet.defaultPasswordFile")
	// FileWalletSignerCacheSize the number of signing keys to keep in memory
	FileWalletSignerCacheSize = ffc("filewallet.signerCacheSize")
	// FileWalletSignerCacheTTL the time to keep an unused signing key in memory
	FileWalletSignerCacheTTL = ffc("filewallet.signerCacheTTL")
	// FileWalletMetadataFormat format to parse the metadata - supported: auto (from extension) / filename / toml / yaml / json (please quote "0x..." strings in YAML)
	FileWalletMetadataFormat = ffc("filewallet.metadata.format")
	// FileWalletMetadataKeyFileProperty use for toml/yaml/json to find the name of the file containing the keystorev3 file
	FileWalletMetadataKeyFileProperty = ffc("filewallet.metadata.keyFileProperty")
	// FileWalletMetadataPasswordFileProperty use for toml/yaml to find the name of the file containing the keystorev3 file
	FileWalletMetadataPasswordFileProperty = ffc("filewallet.metadata.passwordFileProperty")
)

var ServerPrefix config.Prefix

var BackendPrefix config.Prefix

func setDefaults() {
	viper.SetDefault(string(FileWalletEnabled), true)
	viper.SetDefault(string(FileWalletSignerCacheSize), 250)
	viper.SetDefault(string(FileWalletSignerCacheTTL), "24h")
	viper.SetDefault(string(FileWalletMetadataFormat), `auto`)
}

func Reset() {
	config.RootConfigReset(setDefaults)

	ServerPrefix = config.NewPluginConfig("server")
	httpserver.InitHTTPConfPrefix(ServerPrefix, 8545)

	BackendPrefix = config.NewPluginConfig("backend")
	wsclient.InitPrefix(BackendPrefix)
}
