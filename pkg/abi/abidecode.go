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
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

// decodeABIElement is the main entry point to the logic, which looks at the component in hand, and determines
// whether to consume data from the head and/or data depending on the type.
func decodeABIElement(ctx context.Context, breadcrumbs string, block []byte, headStart, headPosition int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {
	switch component.cType {
	case ElementaryComponent:
		// All elementary types consume exactly 32 bytes from the head.
		// Any variable data goes into the data section (calculated as an offset from the headStart)
		cv, err := component.elementaryType.decodeABIData(ctx, breadcrumbs, block, headStart, headPosition, component)
		if err != nil {
			return -1, nil, err
		}
		// So we move the postition beyond the data length of the element
		return 32, cv, err
	case FixedArrayComponent:
		// Fixed arrays are also in the head directly
		headBytesRead, cv, err := decodeABIFixedArrayBytes(ctx, breadcrumbs, block, headStart, headPosition, component)
		if err != nil {
			return -1, nil, err
		}
		return headBytesRead, cv, err
	case DynamicArrayComponent:
		dataOffset, err := decodeABILength(ctx, breadcrumbs, block, headPosition)
		if err != nil {
			return -1, nil, err
		}
		dataOffset = headStart + dataOffset
		// The dynamic array is a new head, starting at its data offset
		cv, err := decodeABIDynamicArrayBytes(ctx, breadcrumbs, block, dataOffset, component)
		if err != nil {
			return -1, nil, err
		}
		return 32, cv, err
	case TupleComponent:
		return walkTupleABIBytes(ctx, breadcrumbs, block, headStart, headPosition, component)
	default:
		return -1, nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, component.cType)
	}

}

func decodeABISignedInt(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv = &ComponentValue{Component: component}
	if headPosition+32 > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}
	cv.Value = ParseInt256TwosComplementBytes(block[headPosition : headPosition+32])
	return cv, err
}

func decodeABIUnsignedInt(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv = &ComponentValue{Component: component}
	if headPosition+32 > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}
	cv.Value = new(big.Int).SetBytes(block[headPosition : headPosition+32])
	return cv, err
}

func intToFixed(ctx context.Context, component *typeComponent, cv *ComponentValue) (*ComponentValue, error) {
	if component.n == 0 {
		return nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, component)
	}
	f := new(big.Float).SetInt(cv.Value.(*big.Int))
	fN := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(component.n)), nil)
	cv.Value = f.Quo(f, new(big.Float).SetInt(fN))
	return cv, nil
}

func decodeABISignedFloat(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABISignedInt(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	return intToFixed(ctx, component, cv)
}

func decodeABIUnsignedFloat(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABIUnsignedInt(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	return intToFixed(ctx, component, cv)
}

func decodeABILength(ctx context.Context, desc string, block []byte, offset int) (count int, err error) {
	if offset+32 > len(block) {
		return -1, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIArrayCount, desc)
	}
	i := new(big.Int).SetBytes(block[offset : offset+32])
	if i.BitLen() > 32 {
		return -1, i18n.NewError(ctx, signermsgs.MsgABIArrayCountTooLarge, i.Text(10), desc)
	}
	return int(i.Int64()), nil
}

func decodeABIBytes(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	var byteLength int
	dataOffset := headPosition
	if component.m == 0 {
		// Variable length bytes. Offset to data in head...
		dataOffset, err = decodeABILength(ctx, desc, block, headPosition)
		if err != nil {
			return nil, err
		}
		dataOffset = headStart + dataOffset
		// ... then array length ahead of the bytes themselves
		byteLength, err = decodeABILength(ctx, desc, block, dataOffset)
		if err != nil {
			return nil, err
		}
		dataOffset += 32
	} else {
		byteLength = int(component.m)
	}
	cv = &ComponentValue{Component: component}
	if dataOffset+byteLength > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}
	b := make([]byte, byteLength)
	copy(b, block[dataOffset:])
	cv.Value = b
	// Byte arrays always consume 32b in the head.
	return cv, err
}

func decodeABIString(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABIBytes(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	cv.Value = string(cv.Value.([]byte))
	return cv, err
}

func decodeABIFixedArrayBytes(ctx context.Context, breadcrumbs string, block []byte, headStart, headPosition int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, component.arrayLength),
	}
	headBytesRead = 0
	// a fixed length sequence of a dynamic type has these parts, using a 'string[2]' type as an example:
	// 0000000000000000000000000000000000000000000000000000000000000020 <= offset of the start of the content
	// 0000000000000000000000000000000000000000000000000000000000000040 <= offset of the 1st element
	// 0000000000000000000000000000000000000000000000000000000000000080 <= offset of the 2nd element
	// 0000000000000000000000000000000000000000000000000000000000000001 <= length of the 1st element
	// 6100000000000000000000000000000000000000000000000000000000000000 <= content of the 1st element
	// 0000000000000000000000000000000000000000000000000000000000000001 <= length of the 2nd element
	// 6200000000000000000000000000000000000000000000000000000000000000 <= content of the 2nd element

	returnOutput := block[headPosition : headPosition+32]

	var content []byte
	if isDynamicType(component.arrayChild) {
		offset := binary.BigEndian.Uint64(returnOutput[len(returnOutput)-8:])
		if offset > uint64(len(block)) {
			return -1, nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIArrayCount, breadcrumbs)
		}
		content = block[offset:]
	} else {
		content = block[headPosition:]
	}

	if headPosition+32*component.arrayLength > len(block) {
		return -1, nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, breadcrumbs)
	}

	offset := 0
	for i := 0; i < component.arrayLength; i++ {
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[fix,i:%d,o:%d]", breadcrumbs, i, headPosition),
			content, headStart, offset, component.arrayChild)
		if err != nil {
			return -1, nil, err
		}
		cv.Children[i] = child
		headBytesRead += childHeadBytes
		headPosition += childHeadBytes
		offset += childHeadBytes
	}
	return headBytesRead, cv, err
}

func isDynamicType(tc *typeComponent) bool {
	switch tc.cType {
	case TupleComponent:
		for _, childType := range tc.tupleChildren {
			if isDynamicType(childType) {
				return true
			}
		}
		return false
	case ElementaryComponent:
		tName := tc.elementaryType.name
		// types of bytes3 are not dynamic types, so make sure to check the suffix
		if tName == "string" || (tName == "bytes" && tc.elementarySuffix == "") {
			return true
		}
		return false
	default:
		return false
	}
}

func decodeABIDynamicArrayBytes(ctx context.Context, breadcrumbs string, block []byte, dataOffset int, component *typeComponent) (cv *ComponentValue, err error) {
	arrayLength, err := decodeABILength(ctx, breadcrumbs, block, dataOffset)
	if err != nil {
		return nil, err
	}
	dataOffset += 32
	dataStart := dataOffset
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, arrayLength),
	}
	for i := 0; i < arrayLength; i++ {
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[dyn,i:%d,b:%d]", breadcrumbs, i, dataOffset),
			block, dataStart, dataOffset, component.arrayChild)
		if err != nil {
			return nil, err
		}
		cv.Children[i] = child
		dataOffset += childHeadBytes
	}
	return cv, err

}

func walkTupleABIBytes(ctx context.Context, breadcrumbs string, block []byte, headStart, headPosition int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, len(component.tupleChildren)),
	}
	headBytesRead = 0
	for i, child := range component.tupleChildren {
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[tup,i:%d,b:%d]", breadcrumbs, i, headPosition),
			block, headStart, headPosition, child)
		if err != nil {
			return -1, nil, err
		}
		cv.Children[i] = child
		headBytesRead += childHeadBytes
		headPosition += childHeadBytes
	}
	return headBytesRead, cv, err
}
