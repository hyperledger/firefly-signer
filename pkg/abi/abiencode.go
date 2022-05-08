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

package abi

import (
	"context"
	"math/big"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

func abiEncodeBytes(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	b, ok := value.([]byte)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "[]byte", value, desc)
	}

	var fixedLength int
	switch tc.elementaryType {
	case ElementaryTypeFunction:
		fixedLength = 24
	default: // ElementaryTypeBytes
		// The type "bytes" (without a length suffix) is a variable encoding
		if tc.elementarySuffix == "" {
			return abiEncodeDynamicBytes(b)
		}
		fixedLength = int(tc.m)
	}

	// Belt and braces length check, although responsibility for generation of all the input data is within this package
	if len(b) < fixedLength || fixedLength > 32 {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgInsufficientDataABIEncode, int(fixedLength), len(b), desc)
	}

	// Copy into the front of a 32byte block, with trailing zeros.
	// That is the head, the data is empty
	data = make([]byte, 32)
	copy(data, b[0:fixedLength])
	return data, false, nil
}

func abiEncodeString(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	s, ok := value.(string)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "string", value, desc)
	}

	// Note we assume UTF-8 encoding has been assured of all input strings. No special handling here.
	return abiEncodeDynamicBytes([]byte(s))
}

func abiEncodeDynamicBytes(value []byte) (data []byte, dynamic bool, err error) {

	dataLen := 32 + // length is prefixed as uint256
		(len(value)/32)*32 // count of whole 32 byte chunks
	if (len(value) % 32) != 0 {
		dataLen += 32 // add 32 byte chunk for remainder
	}
	data = make([]byte, dataLen)
	_ = big.NewInt(int64(len(value))).FillBytes(data[0:32])
	copy(data[32:], value)

	return data, true, nil

}

func abiEncodeSignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	i, ok := value.(*big.Int)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if !checkSignedIntFits(i, tc.m) {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	return serializeInt256TwosComplementBytes(i), false, nil
}

func abiEncodeUnsignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	i, ok := value.(*big.Int)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if i.BitLen() > int(tc.m) {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	data = make([]byte, 32)
	_ = i.FillBytes(data)
	return data, false, nil
}
