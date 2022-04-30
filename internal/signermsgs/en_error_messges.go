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

var ffe = i18n.FFE

//revive:disable
var (
	MsgInvalidOutputType  = ffe("FF20210", "Invalid output type: %s")
	MsgInvalidParam       = ffe("FF20211", "Invalid parameter at position %d for method %s: %s")
	MsgRPCRequestFailed   = ffe("FF20212", "Backend RPC request failed")
	MsgReadDirFile        = ffe("FF20213", "Directory listing failed")
	MsgWalletNotAvailable = ffe("FF20214", "Wallet for address '%s' not available")
	MsgWalletFailed       = ffe("FF20215", "Wallet for address '%s' could not be initialized")
	MsgBadGoTemplate      = ffe("FF20216", "Bad go template for '%s' - try something like '{{ index .signing \"key-file\" }}' syntax")
	MsgNoWalletEnabled    = ffe("FF20217", "No wallets enabled in configuration")
	MsgInvalidRequest     = ffe("FF20218", "Invalid request data")
	MsgInvalidParamCount  = ffe("FF20219", "Invalid number of parameters: expected=%d received=%d")
	MsgMissingFrom        = ffe("FF20220", "Missing 'from' address")
	MsgQueryChainID       = ffe("FF20221", "Failed to query Chain ID")
	MsgSigningFailed      = ffe("FF20222", "Signing failed: %s")
	MsgInvalidTransaction = ffe("FF20223", "Invalid eth_sendTransaction input")
	MsgMissingRequestID   = ffe("FF20224", "Invalid JSON/RPC request. Must set request ID")
)
