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

package signermsgs

import "github.com/hyperledger/firefly/pkg/i18n"

var ffc = i18n.FFC

//revive:disable
var (
	ConfigWalletsKeystoreV3Enabled = ffc("config.wallets.keystorev3.enabled", "Whether the Keystore V3 filesystem wallet is enabled", "boolean")
	ConfigWalletsKeystoreV3Path    = ffc("config.wallets.keystorev3.path", "Path on the filesystem in which Keystore V3 files are located", "string")

	ConfigServerAddress      = ffc("config.server.address", "Local address for the JSON/RPC server to listen on", "string")
	ConfigServerPort         = ffc("config.server.port", "Port for the JSON/RPC server to listen on", "number")
	ConfigAPIPublicURL       = ffc("config.server.publicURL", "External address callers should access API over", "string")
	ConfigServerReadTimeout  = ffc("config.server.readTimeout", "The maximum time to wait when reading from an HTTP connection", "duration")
	ConfigServerWriteTimeout = ffc("config.server.writeTimeout", "The maximum time to wait when writing to a HTTP connection", "duration")

	ConfigBackendURL      = ffc("config.backend.url", "URL for the backend JSON/RPC server / blockchain node", "url")
	ConfigBackendProxyURL = ffc("config.backend.proxy.url", "Optional HTTP proxy URL", "url")
)
