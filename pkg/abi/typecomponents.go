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
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
)

// TypeComponent is a modelled representation of a component of an ABI type.
// We don't just go to the tuple level, we go down all the way through the arrays too.
// This breaks things down into the way in which they are serialized/parsed.
// Example "((uint256,string[2],string[])[][3][],string)" becomes:
// - tuple1
//   - variable size array
//      - fixed size [3] array
//         - variable size array
//            - tuple2
//               - uint256
//               - fixed size [2] array
//                  - string
//               - variable size array
//                   - string
//   - string
// This thus matches the way a JSON structure would exist to supply values in
type TypeComponent interface {
	String() string                     // gives the signature for this type level of the type component hierarchy
	ComponentType() ComponentType       // classification of the component type (tuple, array or elemental)
	ElementaryType() ElementaryTypeInfo // only non-nil for elementary components
	ArrayChild() TypeComponent          // only non-nil for array components
	TupleChildren() []TypeComponent     // only non-nil for tuple components
}

type typeComponent struct {
	cType            ComponentType       // Is this parameter an elementary type, an array, or a tuple
	elementaryType   *elementaryTypeInfo // for elementary types - the type info reference
	elementarySuffix string              // for elementary types - the suffix
	m                uint16              // M dimension of elementary type suffix
	n                uint16              // N dimension of elementary type suffix
	arrayLength      uint32              // The length of a fixed length array
	arrayChild       *typeComponent      // For array parameter
	tupleChildren    []*typeComponent    // For tuple parameters
}

// elementaryTypeInfo defines the string parsing rules, as well as a pointer to the functions for
// serialization to a set of bytes, and back again
type elementaryTypeInfo struct {
	name          string     // The name of the type - the alphabetic characters up to an optional suffix
	suffixType    suffixType // Whether there is a length suffix, and its type
	defaultSuffix string     // If set and there is no suffix supplied, the following suffix is used
	mMin          uint16     // For suffixes with an M dimension, this is the minimum value
	mMax          uint16     // For suffixes with an M dimension, this is the maximum (inclusive) value
	mMod          uint16     // If non-zero, then (M % MMod) == 0 must be true
	nMin          uint16     // For suffixes with an N dimension, this is the minimum value
	nMax          uint16     // For suffixes with an N dimension, this is the maximum (inclusive) value
}

// ElementaryTypeInfo represents the rules for each elementary type understood by this ABI type parser.
type ElementaryTypeInfo interface {
	String() string // gives a summary of the rules the elemental type (used in error reporting)
}

func (et *elementaryTypeInfo) String() string {
	switch et.suffixType {
	case suffixTypeMOptional, suffixTypeMRequired:
		s := fmt.Sprintf("%s<M> (%d <= M <= %d)", et.name, et.mMin, et.mMax)
		if et.mMod != 0 {
			s = fmt.Sprintf("%s (M mod %d == 0)", s, et.mMod)
		}
		if et.suffixType == suffixTypeMOptional {
			s = fmt.Sprintf("%s / %s", et.name, s)
		}
		if et.defaultSuffix != "" {
			s = fmt.Sprintf("%s (%s == %s%s)", s, et.name, et.name, et.defaultSuffix)
		}
		return s
	case suffixTypeMxNRequired:
		s := fmt.Sprintf("%s<M>x<N> (%d <= M <= %d) (%d <= N <= %d)", et.name, et.mMin, et.mMax, et.nMin, et.nMax)
		if et.mMod != 0 {
			s = fmt.Sprintf("%s (M mod %d == 0)", s, et.mMod)
		}
		if et.defaultSuffix != "" {
			s = fmt.Sprintf("%s (%s == %s%s)", s, et.name, et.name, et.defaultSuffix)
		}
		return s
	default:
		return et.name
	}
}

var elementaryTypes = map[string]*elementaryTypeInfo{}

func registerElementaryType(et elementaryTypeInfo) ElementaryTypeInfo {
	elementaryTypes[et.name] = &et
	return &et
}

var (
	ElementaryTypeInt = registerElementaryType(elementaryTypeInfo{
		name:          "int",
		suffixType:    suffixTypeMRequired,
		defaultSuffix: "256",
		mMin:          8,
		mMax:          256,
		mMod:          8,
	})
	ElementaryTypeUint = registerElementaryType(elementaryTypeInfo{
		name:          "uint",
		suffixType:    suffixTypeMRequired,
		defaultSuffix: "256",
		mMin:          8,
		mMax:          256,
		mMod:          8,
	})
	ElementaryTypeAddress = registerElementaryType(elementaryTypeInfo{
		name:       "address",
		suffixType: suffixTypeNone,
	})
	ElementaryTypeBool = registerElementaryType(elementaryTypeInfo{
		name:       "bool",
		suffixType: suffixTypeNone,
	})
	ElementaryTypeFixed = registerElementaryType(elementaryTypeInfo{
		name:          "fixed",
		suffixType:    suffixTypeMxNRequired,
		defaultSuffix: "128x18",
		mMin:          8,
		mMax:          256,
		mMod:          8,
		nMin:          1,
		nMax:          80,
	})
	ElementaryTypeUfixed = registerElementaryType(elementaryTypeInfo{
		name:          "ufixed",
		suffixType:    suffixTypeMxNRequired,
		defaultSuffix: "128x18",
		mMin:          8,
		mMax:          256,
		mMod:          8,
		nMin:          1,
		nMax:          80,
	})
	ElementaryTypeBytes = registerElementaryType(elementaryTypeInfo{
		name:       "bytes",
		suffixType: suffixTypeMOptional, // note that "bytes" without a suffix is a special dynamic sized byte sequence
		mMin:       1,
		mMax:       32,
	})
	ElementaryTypeFunction = registerElementaryType(elementaryTypeInfo{
		name:       "function",
		suffixType: suffixTypeNone,
	})
	ElementaryTypeString = registerElementaryType(elementaryTypeInfo{
		name:       "string",
		suffixType: suffixTypeNone,
	})
	ElementaryTypeTuple = registerElementaryType(elementaryTypeInfo{
		name:       "tuple",
		suffixType: suffixTypeNone,
	})
)

type suffixType int

const (
	suffixTypeNone        suffixType = iota // There is no suffix possible - like "address" or "bool"
	suffixTypeMOptional                     // There is a single dimension suffix, and it is required - like "uin256"
	suffixTypeMRequired                     // There is a single dimension suffix, and it is optional - like "bytes"/"bytes32"
	suffixTypeMxNRequired                   // There is a two-dimensional suffix - like "fixed128x128"
)

type ComponentType int

const (
	ElementaryComponent ComponentType = iota
	FixedArrayComponent
	VariableArrayComponent
	TupleComponent
)

func (tc *typeComponent) String() string {
	switch tc.cType {
	case ElementaryComponent:
		return fmt.Sprintf("%s%s", tc.elementaryType.name, tc.elementarySuffix)
	case FixedArrayComponent:
		return fmt.Sprintf("%s[%d]", tc.arrayChild.String(), tc.arrayLength)
	case VariableArrayComponent:
		return fmt.Sprintf("%s[]", tc.arrayChild.String())
	case TupleComponent:
		buff := new(strings.Builder)
		buff.WriteByte('(')
		for i, child := range tc.tupleChildren {
			if i > 0 {
				buff.WriteByte(',')
			}
			buff.WriteString(child.String())
		}
		buff.WriteByte(')')
		return buff.String()
	default:
		return ""
	}
}

func (tc *typeComponent) ComponentType() ComponentType {
	return tc.cType
}

func (tc *typeComponent) ElementaryType() ElementaryTypeInfo {
	return tc.elementaryType
}

func (tc *typeComponent) ArrayChild() TypeComponent {
	return tc.arrayChild
}

func (tc *typeComponent) TupleChildren() []TypeComponent {
	children := make([]TypeComponent, len(tc.tupleChildren))
	for i, c := range tc.tupleChildren {
		children[i] = c
	}
	return children
}

func (p *Parameter) parseABIParameterComponents(ctx context.Context) (tc *typeComponent, err error) {
	abiTypeString := p.Type

	// Extract the elementary type
	etBuilder := new(strings.Builder)
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
		return nil, i18n.NewError(ctx, i18n.MsgUnsupportedABIType, etStr, abiTypeString)
	}

	// Split what's left of the string into the suffix, and any array definitions
	suffix, arrays := splitElementaryTypeSuffix(abiTypeString, len(etStr))
	if suffix == "" {
		suffix = et.defaultSuffix
	}

	if et == ElementaryTypeTuple {
		tc = &typeComponent{
			cType:         TupleComponent,
			tupleChildren: make([]*typeComponent, len(p.Components)),
		}
		// Process all the components of the tuple
		for i, c := range p.Components {
			if tc.tupleChildren[i], err = c.parseABIParameterComponents(ctx); err != nil {
				return nil, err
			}
		}
	} else {
		tc = &typeComponent{
			cType:            ElementaryComponent,
			elementaryType:   et,
			elementarySuffix: suffix,
		}
		// Process any suffix according to the rules of the elementary type
		switch et.suffixType {
		case suffixTypeNone:
			if suffix != "" {
				return nil, i18n.NewError(ctx, i18n.MsgUnsupportedABISuffix, suffix, abiTypeString, et)
			}
		case suffixTypeMRequired:
			if suffix == "" {
				return nil, i18n.NewError(ctx, i18n.MsgMissingABISuffix, abiTypeString, et)
			}
			if err := parseMSuffix(ctx, abiTypeString, tc, suffix); err != nil {
				return nil, err
			}
		case suffixTypeMOptional:
			if suffix != "" {
				if err := parseMSuffix(ctx, abiTypeString, tc, suffix); err != nil {
					return nil, err
				}
			}
		case suffixTypeMxNRequired:
			if suffix == "" {
				return nil, i18n.NewError(ctx, i18n.MsgMissingABISuffix, abiTypeString, et)
			}
			if err := parseMxNSuffix(ctx, abiTypeString, tc, suffix); err != nil {
				return nil, err
			}
		}
	}

	if arrays != "" {
		// The component needs to be wrapped in some number of array dimensions
		return parseArrays(ctx, abiTypeString, tc, arrays)
	}

	return tc, nil
}

// splitElementaryTypeSuffix splits out the "256" from "[8][]" in "uint256[8][]"
func splitElementaryTypeSuffix(abiTypeString string, pos int) (string, string) {
	suffix := new(strings.Builder)
	for ; pos < len(abiTypeString) && abiTypeString[pos] != '['; pos++ {
		suffix.WriteByte(abiTypeString[pos])
	}
	arrays := new(strings.Builder)
	for ; pos < len(abiTypeString); pos++ {
		arrays.WriteByte(abiTypeString[pos])
	}
	return suffix.String(), arrays.String()
}

// parseMSuffix parses the "256" in "uint256" against the the <M> rules for an elementary type, such as uint<M>, or ufixed<M>x<N>.
func parseMSuffix(ctx context.Context, abiTypeString string, ec *typeComponent, suffix string) error {
	val, err := strconv.ParseUint(suffix, 10, 16)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	ec.m = uint16(val)
	if ec.m < ec.elementaryType.mMin || ec.m > ec.elementaryType.mMax {
		return i18n.NewError(ctx, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	if ec.elementaryType.mMod != 0 && (ec.m%ec.elementaryType.mMod) != 0 {
		return i18n.NewError(ctx, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	return nil
}

// parseNSuffix parses the "18" in "ufixed256x18" against the the <N> rules for an elementary type, such as ufixed<M>x<N>
func parseNSuffix(ctx context.Context, abiTypeString string, ec *typeComponent, suffix string) error {
	val, err := strconv.ParseUint(suffix, 10, 16)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	ec.n = uint16(val)
	if ec.n < ec.elementaryType.nMin || ec.n > ec.elementaryType.nMax {
		return i18n.NewError(ctx, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	return nil
}

// parseMxNSuffix validates the "256x18" in "ufixed256x18", individually validating the <M> and <N> parts of the elementary type
func parseMxNSuffix(ctx context.Context, abiTypeString string, ec *typeComponent, suffix string) error {
	pos := 0
	mStr := new(strings.Builder)
	for ; pos < len(suffix) && suffix[pos] != 'x'; pos++ {
		mStr.WriteByte(suffix[pos])
	}
	if pos >= (len(suffix) - 1) {
		return i18n.NewError(ctx, i18n.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	pos++
	if err := parseMSuffix(ctx, abiTypeString, ec, mStr.String()); err != nil {
		return err
	}
	return parseNSuffix(ctx, abiTypeString, ec, suffix[pos:])
}

// parseArrayM parses the "8" in "uint256[8]" for a fixed length array of <type>[M]
func parseArrayM(ctx context.Context, abiTypeString string, ac *typeComponent, mStr string) error {
	val, err := strconv.ParseUint(mStr, 10, 64)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgInvalidABIArraySpec, abiTypeString)
	}
	ac.arrayLength = uint32(val)
	return nil
}

// parseArrays recursively builds arrays for the "[8][]" part of "uint256[8][]" for variable or fixed array types
func parseArrays(ctx context.Context, abiTypeString string, child *typeComponent, suffix string) (*typeComponent, error) {

	pos := 0
	if pos >= len(suffix) || suffix[pos] != '[' {
		return nil, i18n.NewError(ctx, i18n.MsgInvalidABIArraySpec, abiTypeString)
	}
	mStr := new(strings.Builder)
	for pos++; pos < len(suffix) && suffix[pos] != ']'; pos++ {
		mStr.WriteByte(suffix[pos])
	}
	if pos >= len(suffix) {
		return nil, i18n.NewError(ctx, i18n.MsgInvalidABIArraySpec, abiTypeString)
	}
	pos++
	var ac *typeComponent
	if mStr.Len() == 0 {
		ac = &typeComponent{
			cType:      VariableArrayComponent,
			arrayChild: child,
		}
	} else {
		ac = &typeComponent{
			cType:      FixedArrayComponent,
			arrayChild: child,
		}
		if err := parseArrayM(ctx, abiTypeString, ac, mStr.String()); err != nil {
			return nil, err
		}
	}

	// We might have more dimensions to the array - if so recurse
	if pos < len(suffix) {
		return parseArrays(ctx, abiTypeString, ac, suffix[pos:])
	}

	// We're the last array in the chain
	return ac, nil
}
