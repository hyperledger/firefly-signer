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

package abi

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

/*
TypeComponent is a modelled representation of a component of an ABI type.
We don't just go to the tuple level, we go down all the way through the arrays too.
This breaks things down into the way in which they are serialized/parsed.
Example "((uint256,string[2],string[])[][3][],string)" becomes:
- tuple1
  - variable size array
  - fixed size [3] array
  - variable size array
  - tuple2
  - uint256
  - fixed size [2] array
  - string
  - variable size array
  - string
  - string

This thus matches the way a JSON structure would exist to supply values in
*/
type TypeComponent interface {
	String() string                     // gives the signature for this type level of the type component hierarchy
	ComponentType() ComponentType       // classification of the component type (tuple, array or elemental)
	ElementaryType() ElementaryTypeInfo // only non-nil for elementary components
	ArrayChild() TypeComponent          // only non-nil for array components
	TupleChildren() []TypeComponent     // only non-nil for tuple components
	KeyName() string                    // the name of the ABI property/component, only set for top-level parameters and tuple entries
	Parameter() *Parameter              // the ABI property/component, only set for top-level parameters and tuple entries
	ParseExternal(v interface{}) (*ComponentValue, error)
	ParseExternalCtx(ctx context.Context, v interface{}) (*ComponentValue, error)
	DecodeABIData(d []byte, offset int) (*ComponentValue, error)
	DecodeABIDataCtx(ctx context.Context, d []byte, offest int) (*ComponentValue, error)
}

type typeComponent struct {
	cType            ComponentType       // Is this parameter an elementary type, an array, or a tuple
	elementaryType   *elementaryTypeInfo // for elementary types - the type info reference
	elementarySuffix string              // for elementary types - the suffix
	m                uint16              // M dimension of elementary type suffix
	n                uint16              // N dimension of elementary type suffix
	arrayLength      int                 // The length of a fixed length array
	arrayChild       *typeComponent      // For array parameter
	keyName          string              // For top level ABI entries, and tuple children
	tupleChildren    []*typeComponent    // For tuple parameters
	parameter        *Parameter          // The original ABI parameter for this typeComponent
}

// elementaryTypeInfo defines the string parsing rules, as well as a pointer to the functions for
// serialization to a set of bytes, and back again
type elementaryTypeInfo struct {
	name             string                      // The name of the type - the alphabetic characters up to an optional suffix
	suffixType       suffixType                  // Whether there is a length suffix, and its type
	defaultSuffix    string                      // If set and there is no suffix supplied, the following suffix is used
	defaultM         uint16                      // If the type implicitly has an M value that is not expressed (such as "function")
	mMin             uint16                      // For suffixes with an M dimension, this is the minimum value
	mMax             uint16                      // For suffixes with an M dimension, this is the maximum (inclusive) value
	mMod             uint16                      // If non-zero, then (M % MMod) == 0 must be true
	nMin             uint16                      // For suffixes with an N dimension, this is the minimum value
	nMax             uint16                      // For suffixes with an N dimension, this is the maximum (inclusive) value
	fixed32          bool                        // True if the is at most 32 bytes in length, so directly fits into an event topic
	dynamic          func(c *typeComponent) bool // True if the type is dynamic length
	jsonEncodingType JSONEncodingType            // categorizes how the type can be read/written from input JSON data
	readExternalData func(ctx context.Context, desc string, input interface{}) (interface{}, error)
	encodeABIData    func(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error)
	decodeABIData    func(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error)
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

func (et *elementaryTypeInfo) JSONEncodingType() JSONEncodingType {
	return et.jsonEncodingType
}

var elementaryTypes = map[string]*elementaryTypeInfo{}

func registerElementaryType(et elementaryTypeInfo) ElementaryTypeInfo {
	elementaryTypes[et.name] = &et
	return &et
}

// ElementaryTypeInfo represents the rules for each elementary type understood by this ABI type parser.
// You can do an equality check against the appropriate constant, to check if this is the type you are expecting.
// e.g.
type ElementaryTypeInfo interface {
	String() string                     // gives a summary of the rules the elemental type (used in error reporting)
	JSONEncodingType() JSONEncodingType // categorizes JSON input/output type to one of a small number of options
}

type JSONEncodingType int

const (
	JSONEncodingTypeBool    JSONEncodingType = iota // JSON string or bool
	JSONEncodingTypeInteger                         // JSON string containing integer with/without 0x prefix, or JSON number
	JSONEncodingTypeBytes                           // JSON string containing hex with/without 0x prefix
	JSONEncodingTypeFloat                           // JSON string containing a float with/without 0x prefix or exponent, or JSON number
	JSONEncodingTypeString                          // JSON string containing any unicode characters
)

// tupleTypeString appears in the same place in the ABI as elementary type strings, but it is not an elementary type.
// We treat it separately.
const tupleTypeString = "tuple"

var alwaysFixed = func(tc *typeComponent) bool { return false }

var (
	ElementaryTypeInt = registerElementaryType(elementaryTypeInfo{
		name:             "int",
		suffixType:       suffixTypeMRequired,
		defaultSuffix:    "256",
		mMin:             8,
		mMax:             256,
		mMod:             8,
		fixed32:          true,
		dynamic:          alwaysFixed,
		jsonEncodingType: JSONEncodingTypeInteger,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getIntegerFromInterface(ctx, desc, input)
		},
		encodeABIData: encodeABISignedInteger,
		decodeABIData: decodeABISignedInt,
	})
	ElementaryTypeUint = registerElementaryType(elementaryTypeInfo{
		name:             "uint",
		suffixType:       suffixTypeMRequired,
		defaultSuffix:    "256",
		mMin:             8,
		mMax:             256,
		mMod:             8,
		fixed32:          true,
		dynamic:          alwaysFixed,
		jsonEncodingType: JSONEncodingTypeInteger,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getIntegerFromInterface(ctx, desc, input)
		},
		encodeABIData: encodeABIUnsignedInteger,
		decodeABIData: decodeABIUnsignedInt,
	})
	ElementaryTypeAddress = registerElementaryType(elementaryTypeInfo{
		name:       "address",
		suffixType: suffixTypeNone,
		defaultM:   160, // encoded as "uint160"
		fixed32:    true,
		dynamic:    alwaysFixed,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getUintBytesFromInterface(ctx, desc, input)
		},
		jsonEncodingType: JSONEncodingTypeBytes,
		encodeABIData:    encodeABIUnsignedInteger,
		decodeABIData:    decodeABIUnsignedInt,
	})
	ElementaryTypeBool = registerElementaryType(elementaryTypeInfo{
		name:       "bool",
		suffixType: suffixTypeNone,
		defaultM:   8, // encoded as "uint8"
		fixed32:    true,
		dynamic:    alwaysFixed,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getBoolAsUnsignedIntegerFromInterface(ctx, desc, input)
		},
		jsonEncodingType: JSONEncodingTypeBool,
		encodeABIData:    encodeABIUnsignedInteger,
		decodeABIData:    decodeABIUnsignedInt,
	})
	ElementaryTypeFixed = registerElementaryType(elementaryTypeInfo{
		name:             "fixed",
		suffixType:       suffixTypeMxNRequired,
		defaultSuffix:    "128x18",
		mMin:             8,
		mMax:             256,
		mMod:             8,
		nMin:             1,
		nMax:             80,
		fixed32:          true,
		dynamic:          alwaysFixed,
		jsonEncodingType: JSONEncodingTypeFloat,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getFloatFromInterface(ctx, desc, input)
		},
		encodeABIData: encodeABISignedFloat,
		decodeABIData: decodeABISignedFloat,
	})
	ElementaryTypeUfixed = registerElementaryType(elementaryTypeInfo{
		name:             "ufixed",
		suffixType:       suffixTypeMxNRequired,
		defaultSuffix:    "128x18",
		mMin:             8,
		mMax:             256,
		mMod:             8,
		nMin:             1,
		nMax:             80,
		fixed32:          true,
		dynamic:          alwaysFixed,
		jsonEncodingType: JSONEncodingTypeFloat,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getFloatFromInterface(ctx, desc, input)
		},
		encodeABIData: encodeABIUnsignedFloat,
		decodeABIData: decodeABIUnsignedFloat,
	})
	ElementaryTypeBytes = registerElementaryType(elementaryTypeInfo{
		name:       "bytes",
		suffixType: suffixTypeMOptional, // note that "bytes" without a suffix is a special dynamic sized byte sequence
		mMin:       1,
		mMax:       32,
		fixed32:    false,
		dynamic:    func(tc *typeComponent) bool { return tc.elementarySuffix == "" },
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getBytesFromInterface(ctx, desc, input)
		},
		jsonEncodingType: JSONEncodingTypeBytes,
		encodeABIData:    encodeABIBytes,
		decodeABIData:    decodeABIBytes,
	})
	ElementaryTypeFunction = registerElementaryType(elementaryTypeInfo{
		name:       "function",
		suffixType: suffixTypeNone,
		defaultM:   24, // encoded as "bytes24"
		fixed32:    true,
		dynamic:    alwaysFixed,
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getBytesFromInterface(ctx, desc, input)
		},
		jsonEncodingType: JSONEncodingTypeBytes,
		encodeABIData:    encodeABIBytes,
		decodeABIData:    decodeABIBytes,
	})
	ElementaryTypeString = registerElementaryType(elementaryTypeInfo{
		name:       "string",
		suffixType: suffixTypeNone,
		fixed32:    false,
		dynamic:    func(tc *typeComponent) bool { return true },
		readExternalData: func(ctx context.Context, desc string, input interface{}) (interface{}, error) {
			return getStringFromInterface(ctx, desc, input)
		},
		jsonEncodingType: JSONEncodingTypeString,
		encodeABIData:    encodeABIString,
		decodeABIData:    decodeABIString,
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
	DynamicArrayComponent
	TupleComponent
)

func (tc *typeComponent) String() string {
	switch tc.cType {
	case ElementaryComponent:
		return fmt.Sprintf("%s%s", tc.elementaryType.name, tc.elementarySuffix)
	case FixedArrayComponent:
		return fmt.Sprintf("%s[%d]", tc.arrayChild.String(), tc.arrayLength)
	case DynamicArrayComponent:
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

func (tc *typeComponent) KeyName() string {
	return tc.keyName
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

func (tc *typeComponent) ParseExternal(input interface{}) (*ComponentValue, error) {
	return tc.ParseExternalCtx(context.Background(), input)
}

func (tc *typeComponent) ParseExternalCtx(ctx context.Context, input interface{}) (*ComponentValue, error) {
	return tc.parseExternal(ctx, "", input)
}

func (tc *typeComponent) DecodeABIData(b []byte, offset int) (*ComponentValue, error) {
	return tc.DecodeABIDataCtx(context.Background(), b, offset)
}

func (tc *typeComponent) DecodeABIDataCtx(ctx context.Context, b []byte, offset int) (*ComponentValue, error) {
	if tc.cType != TupleComponent {
		return nil, i18n.NewError(ctx, signermsgs.MsgDecodeNotTuple, tc.cType)
	}
	_, cv, err := walkTupleABIBytes(ctx, b, offset, tc)
	return cv, err
}

func (tc *typeComponent) Parameter() *Parameter {
	return tc.parameter
}

func (tc *typeComponent) parseExternal(ctx context.Context, desc string, input interface{}) (*ComponentValue, error) {
	return walkInput(ctx, desc, input, tc)
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

	// Split what's left of the string into the suffix, and any array definitions
	suffix, arrays := splitElementaryTypeSuffix(abiTypeString, len(etStr))

	if etStr == tupleTypeString {
		tc = &typeComponent{
			cType:         TupleComponent,
			tupleChildren: make([]*typeComponent, len(p.Components)),
			keyName:       p.Name,
			parameter:     p,
		}
		// Process all the components of the tuple
		for i, c := range p.Components {
			if tc.tupleChildren[i], err = c.parseABIParameterComponents(ctx); err != nil {
				return nil, err
			}
		}
	} else {
		et, ok := elementaryTypes[etStr]
		if !ok {
			return nil, i18n.NewError(ctx, signermsgs.MsgUnsupportedABIType, etStr, abiTypeString)
		}
		if suffix == "" {
			suffix = et.defaultSuffix
		}
		tc = &typeComponent{
			cType:            ElementaryComponent,
			elementaryType:   et,
			elementarySuffix: suffix,
			keyName:          p.Name,
			m:                et.defaultM,
			parameter:        p,
		}
		// Process any suffix according to the rules of the elementary type
		switch et.suffixType {
		case suffixTypeNone:
			if suffix != "" {
				return nil, i18n.NewError(ctx, signermsgs.MsgUnsupportedABISuffix, suffix, abiTypeString, et)
			}
		case suffixTypeMRequired:
			if suffix == "" {
				return nil, i18n.NewError(ctx, signermsgs.MsgMissingABISuffix, abiTypeString, et)
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
				return nil, i18n.NewError(ctx, signermsgs.MsgMissingABISuffix, abiTypeString, et)
			}
			if err := parseMxNSuffix(ctx, abiTypeString, tc, suffix); err != nil {
				return nil, err
			}
		}
	}

	if arrays != "" {
		// The component needs to be wrapped in some number of array dimensions
		return p.parseArrays(ctx, abiTypeString, tc, arrays, p.Name)
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
		return i18n.WrapError(ctx, err, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	ec.m = uint16(val)
	if ec.m < ec.elementaryType.mMin || ec.m > ec.elementaryType.mMax {
		return i18n.NewError(ctx, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	if ec.elementaryType.mMod != 0 && (ec.m%ec.elementaryType.mMod) != 0 {
		return i18n.NewError(ctx, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	return nil
}

// parseNSuffix parses the "18" in "ufixed256x18" against the the <N> rules for an elementary type, such as ufixed<M>x<N>
func parseNSuffix(ctx context.Context, abiTypeString string, ec *typeComponent, suffix string) error {
	val, err := strconv.ParseUint(suffix, 10, 16)
	if err != nil {
		return i18n.WrapError(ctx, err, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
	}
	ec.n = uint16(val)
	if ec.n < ec.elementaryType.nMin || ec.n > ec.elementaryType.nMax {
		return i18n.NewError(ctx, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
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
		return i18n.NewError(ctx, signermsgs.MsgInvalidABISuffix, abiTypeString, ec.elementaryType)
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
		return i18n.WrapError(ctx, err, signermsgs.MsgInvalidABIArraySpec, abiTypeString)
	}
	ac.arrayLength = int(val)
	return nil
}

// parseArrays recursively builds arrays for the "[8][]" part of "uint256[8][]" for variable or fixed array types
func (p *Parameter) parseArrays(ctx context.Context, abiTypeString string, child *typeComponent, suffix, keyName string) (*typeComponent, error) {

	pos := 0
	if pos >= len(suffix) || suffix[pos] != '[' {
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidABIArraySpec, abiTypeString)
	}
	mStr := new(strings.Builder)
	for pos++; pos < len(suffix) && suffix[pos] != ']'; pos++ {
		mStr.WriteByte(suffix[pos])
	}
	if pos >= len(suffix) {
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidABIArraySpec, abiTypeString)
	}
	pos++
	var ac *typeComponent
	if mStr.Len() == 0 {
		ac = &typeComponent{
			cType:      DynamicArrayComponent,
			arrayChild: child,
			keyName:    keyName,
			parameter:  p,
		}
	} else {
		ac = &typeComponent{
			cType:      FixedArrayComponent,
			arrayChild: child,
			keyName:    keyName,
			parameter:  p,
		}
		if err := parseArrayM(ctx, abiTypeString, ac, mStr.String()); err != nil {
			return nil, err
		}
	}

	// We might have more dimensions to the array - if so recurse
	if pos < len(suffix) {
		return p.parseArrays(ctx, abiTypeString, ac, suffix[pos:], keyName)
	}

	// We're the last array in the chain
	return ac, nil
}
