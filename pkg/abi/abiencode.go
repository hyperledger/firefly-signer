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

func abiEncodeBytes(ctx context.Context, desc string, tc *typeComponent, value interface{}) (head, tail []byte, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	b, ok := value.([]byte)
	if !ok {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "[]byte", value, desc)
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
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInsufficientDataABIEncode, int(fixedLength), len(b), desc)
	}

	// Copy into the front of a 32byte block, with trailing zeros.
	// That is the head, the tail is empty
	head = make([]byte, 32)
	copy(head, b[0:fixedLength])
	tail = []byte{}
	return head, tail, nil
}

func abiEncodeString(ctx context.Context, desc string, tc *typeComponent, value interface{}) (head, tail []byte, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	s, ok := value.(string)
	if !ok {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "string", value, desc)
	}

	// Note we assume UTF-8 encoding has been assured of all input strings. No special handling here.
	return abiEncodeDynamicBytes([]byte(s))
}

func abiEncodeDynamicBytes(value []byte) (head, tail []byte, err error) {

	// Head is big-endian left padded 32byte integer of the length of the byte string
	head = make([]byte, 32)
	head = big.NewInt(int64(len(value))).FillBytes(head)

	// Tail is the actual byte-string, padded ot a multiple of 32
	tailLen := (len(value) / 32) * 32
	if (len(value) % 32) != 0 {
		tailLen += 32
	}
	tail = make([]byte, tailLen)
	copy(tail, value)

	return head, tail, nil

}

func abiEncodeSignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (head, tail []byte, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	i, ok := value.(*big.Int)
	if !ok {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if !checkSignedIntFits(i, tc.m) {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	head = serializeInt256TwosComplementBytes(i)
	tail = []byte{}
	return head, tail, nil
}

func abiEncodeUnsignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (head, tail []byte, err error) {
	// Belt and braces type check, although responsibility for generation of all the input data is within this package
	i, ok := value.(*big.Int)
	if !ok {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if i.BitLen() > int(tc.m) {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	head = make([]byte, 32)
	head = i.FillBytes(head)
	tail = []byte{}
	return head, tail, nil
}
