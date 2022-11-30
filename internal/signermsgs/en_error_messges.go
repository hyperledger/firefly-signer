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

import (
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"golang.org/x/text/language"
)

var ffe = func(key, translation string, statusHint ...int) i18n.ErrorMessageKey {
	return i18n.FFE(language.AmericanEnglish, key, translation, statusHint...)
}

//revive:disable
var (
	MsgInvalidOutputType           = ffe("FF22010", "Invalid output type: %s")
	MsgInvalidParam                = ffe("FF22011", "Invalid parameter at position %d for method %s: %s")
	MsgRPCRequestFailed            = ffe("FF22012", "Backend RPC request failed: %s")
	MsgReadDirFile                 = ffe("FF22013", "Directory listing failed")
	MsgWalletNotAvailable          = ffe("FF22014", "Wallet for address '%s' not available", 404)
	MsgWalletFailed                = ffe("FF22015", "Wallet for address '%s' could not be initialized")
	MsgBadGoTemplate               = ffe("FF22016", "Bad go template for '%s' - try something like '{{ index .signing \"key-file\" }}' syntax")
	MsgNoWalletEnabled             = ffe("FF22017", "No wallets enabled in configuration")
	MsgInvalidRequest              = ffe("FF22018", "Invalid request data")
	MsgInvalidParamCount           = ffe("FF22019", "Invalid number of parameters: expected=%d received=%d")
	MsgMissingFrom                 = ffe("FF22020", "Missing 'from' address")
	MsgQueryChainID                = ffe("FF22021", "Failed to query Chain ID")
	MsgSigningFailed               = ffe("FF22022", "Signing failed: %s")
	MsgInvalidTransaction          = ffe("FF22023", "Invalid eth_sendTransaction input")
	MsgMissingRequestID            = ffe("FF22024", "Invalid JSON/RPC request. Must set request ID")
	MsgUnsupportedABIType          = ffe("FF22025", "Unsupported elementary type '%s' in ABI type '%s'")
	MsgUnsupportedABISuffix        = ffe("FF22026", "Unsupported type suffix '%s' in ABI type '%s' - expected %s")
	MsgMissingABISuffix            = ffe("FF22027", "Missing type suffix in ABI type '%s' - expected %s")
	MsgInvalidABISuffix            = ffe("FF22028", "Invalid suffix in ABI type '%s' - expected %s")
	MsgInvalidABIArraySpec         = ffe("FF22029", "Invalid array suffix in ABI type '%s'")
	MsgInvalidIntegerABIInput      = ffe("FF22030", "Unable to parse '%v' of type %T as integer for component %s")
	MsgInvalidFloatABIInput        = ffe("FF22031", "Unable to parse '%v' of type %T as floating point number for component %s")
	MsgInvalidStringABIInput       = ffe("FF22032", "Unable to parse '%v' of type %T as string for component %s")
	MsgInvalidBoolABIInput         = ffe("FF22033", "Unable to parse '%v' of type %T as boolean for component %s")
	MsgInvalidHexABIInput          = ffe("FF22034", "Unable to parse input of type %T as hex for component %s")
	MsgMustBeSliceABIInput         = ffe("FF22035", "Unable to parse input of type %T for component %s - must be an array")
	MsgFixedLengthABIArrayMismatch = ffe("FF22036", "Input array is length %d, and required fixed array length is %d for component %s")
	MsgTupleABIArrayMismatch       = ffe("FF22037", "Input array is length %d, and required tuple component count is %d for component %s")
	MsgTupleABINotArrayOrMap       = ffe("FF22038", "Input type %T is not array or map for component %s")
	MsgTupleInABINoName            = ffe("FF22039", "Tuple child %d does not have a name for component %s")
	MsgMissingInputKeyABITuple     = ffe("FF22040", "Input map missing key '%s' required for tuple component %s")
	MsgBadABITypeComponent         = ffe("FF22041", "Bad ABI type component: %d")
	MsgWrongTypeComponentABIEncode = ffe("FF22042", "Incorrect type expected=%s found=%T for ABI encoding of component %s")
	MsgInsufficientDataABIEncode   = ffe("FF22043", "Insufficient data elements on input expected=%d found=%d for ABI encoding of component %s")
	MsgNumberTooLargeABIEncode     = ffe("FF22044", "Numeric value does not fit in bit length %d for ABI encoding of component %s")
	MsgNotEnoughBytesABIArrayCount = ffe("FF22045", "Insufficient bytes to read array index for component %s")
	MsgABIArrayCountTooLarge       = ffe("FF22046", "Array index %s too large for component %s")
	MsgNotEnoughBytesABIValue      = ffe("FF22047", "Insufficient bytes to read %s value %s")
	MsgNotEnoughBytesABISignature  = ffe("FF22048", "Insufficient bytes to read signature")
	MsgIncorrectABISignatureID     = ffe("FF22049", "Incorrect ID for signature %s expected=%s found=%s")
	MsgUnknownABIElementaryType    = ffe("FF22050", "Unknown elementary type %s for component %s")
	MsgUnknownTupleSerializer      = ffe("FF22051", "Unknown tuple serialization option %d")
	MsgInvalidFFIDetailsSchema     = ffe("FF22052", "Invalid FFI details schema for '%s'")
	MsgEventsInsufficientTopics    = ffe("FF22053", "Ran out of topics for indexed fields at field %d of %s")
	MsgEventSignatureMismatch      = ffe("FF22054", "Event signature mismatch for '%s': expected='%s' found='%s'")
	MsgFFITypeMismatch             = ffe("FF22055", "Input type '%s' is not valid for ABI type '%s'")
	MsgBadRegularExpression        = ffe("FF22056", "Bad regular expression for /%s/: %s")
	MsgMissingRegexpCaptureGroup   = ffe("FF22057", "Regular expression is missing a capture group (subexpression) for address: /%s/")
	MsgAddressMismatch             = ffe("FF22059", "Address '%s' loaded from wallet file does not match requested lookup address / filename '%s'")
	MsgFailedToStartListener       = ffe("FF22060", "Failed to start filesystem listener: %s")
	MsgDecodeNotTuple              = ffe("FF22061", "Decode can only be called against a root tuple component type=%d")
)
