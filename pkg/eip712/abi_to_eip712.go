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

package eip712

import (
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

var internalTypeStructExtractor = regexp.MustCompile(`^struct (.*\.)?([^.]+)$`)

// EIP-712 hashStruct() implementation using abi.ComponentValue as an input definition
//
// This file takes ComponentValue trees from the ABI encoding package,
// and traverses them into EIP-712 encodable structures according to
// the rules laid out in:
//
//	https://eips.ethereum.org/EIPS/eip-712
func HashStructABI(ctx context.Context, v *abi.ComponentValue) (ethtypes.HexBytes0xPrefix, error) {
	encodedType, err := EncodeTypeABI(ctx, v.Component)
	if err != nil {
		return nil, err
	}

	encodedData, err := EncodeDataABI(ctx, v)
	if err != nil {
		return nil, err
	}

	// typeHash = keccak256(encodeType(typeOf(s)))
	typeHash := hashString(encodedType)
	/// hashStruct(s : 𝕊) = keccak256(typeHash ‖ encodeData(s))
	hash := sha3.NewLegacyKeccak256()
	hash.Write(typeHash)
	hash.Write([]byte(encodedData))
	return hash.Sum(nil), nil
}

func hashString(s string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(s))
	return hash.Sum(nil)
}

// EIP-712 encodeType() implementation using abi.TypeComponent as an input definition
//
// Builds the map of struct names to types
func EncodeTypeABI(ctx context.Context, tc abi.TypeComponent) (string, error) {
	typeSet, err := buildTypeSetABI(ctx, tc)
	if err != nil {
		return "", err
	}
	return typeSet.Encode(tc.Parameter().Name), nil
}

// EIP-712 encodeData() implementation using abi.ComponentValue as an input definition
//
// Recurses into the structure following the rules of EIP-712 to construct the encoded hash.
func EncodeDataABI(ctx context.Context, v *abi.ComponentValue) (ethtypes.HexBytes0xPrefix, error) {
	tc := v.Component
	switch tc.ComponentType() {
	case abi.TupleComponent, abi.DynamicArrayComponent, abi.FixedArrayComponent:
		// Concatenate an encoding of each component
		buff := new(bytes.Buffer)
		for _, child := range v.Children {
			childData, err := EncodeDataABI(ctx, child)
			if err != nil {
				return nil, err
			}
			buff.Write(childData)
		}
		return buff.Bytes(), nil
	case abi.ElementaryComponent:
		return encodeElementaryDataABI(ctx, v)
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712UnknownABICompType, tc.ComponentType())
	}
}

func buildTypeSetABI(ctx context.Context, tc abi.TypeComponent) (TypeSet, error) {
	if tc.ComponentType() != abi.TupleComponent {
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712PrimaryNotTuple, tc.String())
	}
	// First we need to build the sorted array of types for `encodeType`
	typeSet := make(TypeSet)
	if err := addABITypes(ctx, tc, typeSet); err != nil {
		return nil, err
	}
	return typeSet, nil
}

// Maps a parsed ABI component type to an EIP-712 type string
// - Subset of elementary types, with aliases resolved
// - Struct types are simply the name of the type
// - Fixed and dynamic array suffixes are supported
func mapABIType(ctx context.Context, tc abi.TypeComponent) (string, error) {
	switch tc.ComponentType() {
	case abi.TupleComponent:
		return tc.Parameter().Type, nil
	case abi.DynamicArrayComponent, abi.FixedArrayComponent:
		child, err := mapABIType(ctx, tc.ArrayChild())
		if err != nil {
			return "", err
		}
		if tc.ComponentType() == abi.FixedArrayComponent {
			return fmt.Sprintf("%s[%d]", child, tc.FixedArrayLen()), nil
		}
		return child + "[]", nil
	default:
		return mapElementaryABIType(ctx, tc)
	}
}

// Maps one of the parsed ABI elementary types to an EIP-712 elementary type
func mapElementaryABIType(ctx context.Context, tc abi.TypeComponent) (string, error) {
	et := tc.ElementaryType()
	if et == nil {
		return "", i18n.NewError(ctx, signermsgs.MsgEIP712NotElementary, tc)
	}
	switch et.BaseType() {
	case abi.BaseTypeAddress, abi.BaseTypeBool, abi.BaseTypeString:
		// Types that need no transposition
		return string(et.BaseType()), nil
	case abi.BaseTypeInt, abi.BaseTypeUInt:
		// Types with supported suffixes - note ABI package sorts alias resolution for us
		return string(et.BaseType()) + tc.ElementarySuffix(), nil
	case abi.BaseTypeBytes:
		// Bytes is special
		if tc.ElementaryFixed() {
			return string(et.BaseType()) + tc.ElementarySuffix(), nil
		}
		return string(et.BaseType()), nil
	default:
		// EIP-712 does not support the other types
		return "", i18n.NewError(ctx, signermsgs.MsgEIP712UnsupportedABIType, tc)
	}
}

func encodeElementaryDataABI(ctx context.Context, v *abi.ComponentValue) (ethtypes.HexBytes0xPrefix, error) {
	tc := v.Component
	et := tc.ElementaryType()
	if et == nil {
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712NotElementary, tc)
	}
	switch et.BaseType() {
	case abi.BaseTypeAddress, abi.BaseTypeBool, abi.BaseTypeInt, abi.BaseTypeUInt:
		// Types that need no transposition
		b, _, err := v.ElementaryABIDataCtx(ctx)
		return b, err
	case abi.BaseTypeBytes:
		if tc.ElementaryFixed() {
			// If it's bytes1 -> bytes32
			b, _, err := v.ElementaryABIDataCtx(ctx)
			return b, err
		}
		b, ok := v.Value.([]byte)
		if !ok {
			return nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "[]byte", v.Value, v.Component.KeyName())
		}
		return b, nil
	case abi.BaseTypeString:
		s, ok := v.Value.(string)
		if !ok {
			return nil, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "string", v.Value, v.Component.KeyName())
		}
		return []byte(s), nil
	default:
		// EIP-712 does not support the other types
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712UnsupportedABIType, tc)
	}
}

// ABI does not formally contain the Struct name - as it's not required for encoding the value.
// EIP-712 requires the Struct name as it is used through the standard, including to de-dup definitions
//
// Solidity uses the "internalType" field by convention as an extension to ABI, so we require
// that here for EIP-712 encoding to be successful.
func extractSolidityTypeName(ctx context.Context, param *abi.Parameter) (string, error) {
	match := internalTypeStructExtractor.FindStringSubmatch(param.InternalType)
	if match == nil {
		return "", i18n.NewError(ctx, signermsgs.MsgEIP712BadInternalType, param.InternalType)
	}
	return match[2], nil
}

// Recursively find all types, with a name -> encoded name map.
func addABITypes(ctx context.Context, tc abi.TypeComponent, typeSet TypeSet) error {
	switch tc.ComponentType() {
	case abi.TupleComponent:
		typeName, err := extractSolidityTypeName(ctx, tc.Parameter())
		if err != nil {
			return err
		}

		if _, mapped := typeSet[typeName]; mapped {
			// we've already mapped this type
			return nil
		}
		t := make(Type, len(tc.TupleChildren()))
		for i, child := range tc.TupleChildren() {
			ts, err := mapABIType(ctx, child)
			if err != nil {
				return err
			}
			t[i] = &TypeMember{
				Name: child.KeyName(),
				Type: ts,
			}
		}
		typeSet[typeName] = t
		// recurse
		for _, child := range tc.TupleChildren() {
			if err := addABITypes(ctx, child, typeSet); err != nil {
				return err
			}
		}
		return nil
	case abi.DynamicArrayComponent, abi.FixedArrayComponent:
		// recurse into the child
		return addABITypes(ctx, tc.ArrayChild(), typeSet)
	default:
		// from a type collection perspective, this is a leaf - nothing to do
		return nil
	}
}
