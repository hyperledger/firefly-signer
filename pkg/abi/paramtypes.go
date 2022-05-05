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
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
)

// ParsedType is a modelled representation of an ABI type, that
type ParameterType struct {
	ElementaryType ElementaryType // The first characters that all variations of this type have "uint" or "bool"
	LengthSuffix   LengthSuffix   // Whether there is a length suffix, and its type
	ArrayType      ArrayType      // Whether this is an array, and the sub-type of array
	ArrayItem      *ParameterType // For array types, this is the thing that the array is of
	ArraySize      int            // If this is a fixed array, the size of the array
	Tuple          bool           // Whether this is a tuple - which is like an array where each element can have a different type
	MMin           int            // For suffixes with an M dimension, this is the minimum value
	MMax           int            // For suffixes with an M dimension, this is the maximum (inclusive) value
	NMin           int            // For suffixes with an N dimension, this is the minimum value
	NMax           int            // For suffixes with an N dimension, this is the maximum (inclusive) value
}

type ElementaryType string

var elementaryTypes = map[string]ElementaryType{}

func elementaryType(s string) ElementaryType {
	elementaryTypes[s] = ElementaryType(s)
	return ElementaryType(s)
}

var (
	ElementaryTypeInt      = elementaryType("int")
	ElementaryTypeUint     = elementaryType("uint")
	ElementaryTypeAddress  = elementaryType("address")
	ElementaryTypeBool     = elementaryType("bool")
	ElementaryTypeFixed    = elementaryType("fixed")
	ElementaryTypeUfixed   = elementaryType("ufixed")
	ElementaryTypeBytes    = elementaryType("bytes")
	ElementaryTypeString   = elementaryType("string")
	ElementaryTypeFunction = elementaryType("function")
	ElementaryTypeTuple    = elementaryType("tuple")
)

type LengthSuffix int

const (
	LengthSuffixNone LengthSuffix = iota // There is no suffix possible - like "address" or "bool"
	LengthSuffixM                        // There is a single dimension suffix - like "uin256"
	LengthSuffixMxN                      // There is a two-dimensional suffix - like "fixed128x128"
)

type ArrayType int

const (
	ArrayTypeNone     ArrayType = iota // Not an array
	ArrayTypeFixed                     // Fixed length arrays
	ArrayTypeVariable                  // Variable length arrays
)

func ParseParameterType(ctx context.Context, abiTypeString string) (*ParameterType, error) {

	// Extract the elementary type
	etBuilder := new(strings.Builder)
	etBuilder.Grow(len(abiTypeString))
	for _, r := range abiTypeString {
		if r >= 'a' && r <= 'z' {
			etBuilder.WriteRune(r)
		} else {
			break
		}
	}
	etStr := etBuilder.String()
	et, ok := elementaryTypes[etStr]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgInvalidElementaryABIType, etStr, abiTypeString)
	}
	pt := &ParameterType{
		ElementaryType: et,
	}
	switch {

	case len(abiTypeString) == len(etStr):
		return pt, nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgInvalidElementaryABIType, abiTypeString[len(etStr):], abiTypeString)
	}

}
