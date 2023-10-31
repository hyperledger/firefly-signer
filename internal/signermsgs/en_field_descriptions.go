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

package signermsgs

var (
	ABIEntryAnonymous       = ffm("EthABIEntry.anonymous", "If the event is anonymous then the signature of the event does not take up a topic slot")
	ABIEntryType            = ffm("EthABIEntry.type", "The type of the ABI entry: 'event', 'error', 'function', 'constructor', 'receive', or 'fallback'")
	ABIEntryName            = ffm("EthABIEntry.name", "The name of the ABI entry")
	ABIEntryInputs          = ffm("EthABIEntry.inputs", "Array of ABI parameter definitions")
	ABIEntryOutputs         = ffm("EthABIEntry.outputs", "Array of ABI parameter definitions")
	ABIEntryStateMutability = ffm("EthABIEntry.stateMutability", "The state mutability of the function: 'pure', 'view', 'nonpayable' (the default) and 'payable'")
	ABIEntryConstant        = ffm("EthABIEntry.constant", "Functions only: Superseded by stateMutability payable/nonpayable")
	ABIEntryPayable         = ffm("EthABIEntry.payable", "Functions only: Superseded by stateMutability pure/view")

	ABIParameterName         = ffm("EthABIParameter.name", "The name of the parameter")
	ABIParameterType         = ffm("EthABIParameter.type", "The type of the parameter per the ABI specification")
	ABIParameterComponents   = ffm("EthABIParameter.components", "An array of components, if the parameter is a tuple")
	ABIParameterIndexed      = ffm("EthABIParameter.indexed", "Whether this parameter uses one of the topics, or is in the data area")
	ABIParameterInternalType = ffm("EthABIParameter.internalType", "Used by the solc compiler to include additional details - importantly the struct name for tuples")
)
