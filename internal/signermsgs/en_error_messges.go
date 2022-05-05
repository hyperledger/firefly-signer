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

import "github.com/hyperledger/firefly-common/pkg/i18n"

var ffe = i18n.FFE

//revive:disable
var (
	MsgInvalidOutputType  = ffe("FF22010", "Invalid output type: %s")
	MsgInvalidParam       = ffe("FF22011", "Invalid parameter at position %d for method %s: %s")
	MsgRPCRequestFailed   = ffe("FF22012", "Backend RPC request failed")
	MsgReadDirFile        = ffe("FF22013", "Directory listing failed")
	MsgWalletNotAvailable = ffe("FF22014", "Wallet for address '%s' not available")
	MsgWalletFailed       = ffe("FF22015", "Wallet for address '%s' could not be initialized")
	MsgBadGoTemplate      = ffe("FF22016", "Bad go template for '%s' - try something like '{{ index .signing \"key-file\" }}' syntax")
	MsgNoWalletEnabled    = ffe("FF22017", "No wallets enabled in configuration")
	MsgInvalidRequest     = ffe("FF22018", "Invalid request data")
	MsgInvalidParamCount  = ffe("FF22019", "Invalid number of parameters: expected=%d received=%d")
	MsgMissingFrom        = ffe("FF22020", "Missing 'from' address")
	MsgQueryChainID       = ffe("FF22021", "Failed to query Chain ID")
	MsgSigningFailed      = ffe("FF22022", "Signing failed: %s")
	MsgInvalidTransaction = ffe("FF22023", "Invalid eth_sendTransaction input")
	MsgMissingRequestID   = ffe("FF22024", "Invalid JSON/RPC request. Must set request ID")
)
