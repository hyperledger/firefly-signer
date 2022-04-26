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
	"github.com/spf13/viper"
)

var ffc = config.AddRootKey

var (
	// WalletsKeystoreV3Enabled if the Keystore V3 wallet is enabled
	WalletsKeystoreV3Enabled = ffc("wallets.keystorev3.enabled")
	// WalletsKeystoreV3Path the path of the Keystore V3 wallet path
	WalletsKeystoreV3Path = ffc("wallets.keystorev3.path")
)

var ConnectorPrefix config.Prefix

var FFCorePrefix config.Prefix

var APIPrefix config.Prefix

var PolicyEngineBasePrefix config.Prefix

func setDefaults() {
	viper.SetDefault(string(WalletsKeystoreV3Enabled), true)
}

func Reset() {
	config.RootConfigReset(setDefaults)
}
