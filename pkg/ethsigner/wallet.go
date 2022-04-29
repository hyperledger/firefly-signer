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

package ethsigner

import (
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
)

// Wallet is the common interface can be implemented across wallet/signing capabilities
type Wallet interface {
	Sign(addr ethtypes.Address, tx *Transaction, chainID int64) ([]byte, error)
	// SignPrivateTxn(addr ethtypes.Address, ptx *Transaction, chainID int64) ([]byte, error)
	Initialize() error
	GetAccounts() []ethtypes.Address
	Refresh() error
	Close() error
}
