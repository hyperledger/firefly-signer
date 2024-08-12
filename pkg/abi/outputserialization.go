// Copyright © 2024 Kaleido, Inc.
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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

// Serializer contains a set of options for how to serialize an parsed
// ABI value tree, into JSON.
type Serializer struct {
	ts FormattingMode
	is IntSerializer
	fs FloatSerializer
	bs ByteSerializer
	dn DefaultNameGenerator
}

// NewSerializer creates a new ABI value tree serializer, with the default
// configuration.
// - FormattingMode: FormatAsObjects
// - IntSerializer: DecimalStringIntSerializer
// - FloatSerializer: DecimalStringFloatSerializer
// - ByteSerializer: HexByteSerializer
func NewSerializer() *Serializer {
	return &Serializer{
		is: Base10StringIntSerializer,
		fs: Base10StringFloatSerializer,
		bs: HexByteSerializer,
		dn: NumericDefaultNameGenerator,
	}
}

// FormattingMode affects how function parameters, and child tuples, are serialized.
type FormattingMode int

const (
	// FormatAsObjects uses the names of the function / event / tuple parameters as keys in an object
	FormatAsObjects FormattingMode = iota
	// FormatAsFlatArrays uses flat arrays of flat values
	FormatAsFlatArrays
	// FormatAsSelfDescribingArrays uses arrays of structures with {"name":"arg1","type":"uint256","value":...}
	FormatAsSelfDescribingArrays
)

var (
	maxSafeJSONNumberInt   = big.NewInt(9007199254740991)
	maxSafeJSONNumberFloat = big.NewFloat(9007199254740991)
	minSafeJSONNumberInt   = big.NewInt(-9007199254740991)
	minSafeJSONNumberFloat = big.NewFloat(-9007199254740991)
)

type DefaultNameGenerator func(idx int) string

type IntSerializer func(i *big.Int) interface{}

type FloatSerializer func(f *big.Float) interface{}

type ByteSerializer func(b []byte) interface{}

func (s *Serializer) SetFormattingMode(ts FormattingMode) *Serializer {
	s.ts = ts
	return s
}

func (s *Serializer) SetIntSerializer(is IntSerializer) *Serializer {
	s.is = is
	return s
}

func (s *Serializer) SetFloatSerializer(fs FloatSerializer) *Serializer {
	s.fs = fs
	return s
}

func (s *Serializer) SetByteSerializer(bs ByteSerializer) *Serializer {
	s.bs = bs
	return s
}

func (s *Serializer) SetDefaultNameGenerator(dn DefaultNameGenerator) *Serializer {
	s.dn = dn
	return s
}

func Base10StringIntSerializer(i *big.Int) interface{} {
	return i.String()
}

func HexIntSerializer0xPrefix(i *big.Int) interface{} {
	return "0x" + i.Text(16)
}

func Base10StringFloatSerializer(f *big.Float) interface{} {
	return f.String()
}

func NumberIfFitsOrBase10StringFloatSerializer(f *big.Float) interface{} {
	if f.Cmp(maxSafeJSONNumberFloat) > 0 ||
		f.Cmp(minSafeJSONNumberFloat) < 0 {
		return f.String()
	}
	v, _ := f.Float64()
	return v
}

func NumberIfFitsOrBase10StringIntSerializer(i *big.Int) interface{} {
	if i.Cmp(maxSafeJSONNumberInt) > 0 ||
		i.Cmp(minSafeJSONNumberInt) < 0 {
		return i.String()
	}
	return float64(i.Int64())
}

func HexByteSerializer(b []byte) interface{} {
	return hex.EncodeToString(b)
}

func HexByteSerializer0xPrefix(b []byte) interface{} {
	return "0x" + hex.EncodeToString(b)
}

func NumericDefaultNameGenerator(idx int) string {
	return strconv.FormatInt(int64(idx), 10)
}

func (s *Serializer) SerializeInterface(cv *ComponentValue) (interface{}, error) {
	return s.SerializeInterfaceCtx(context.Background(), cv)
}

func (s *Serializer) SerializeInterfaceCtx(ctx context.Context, cv *ComponentValue) (interface{}, error) {
	return s.walkOutput(ctx, "", cv)
}

func (s *Serializer) SerializeJSON(cv *ComponentValue) ([]byte, error) {
	return s.SerializeJSONCtx(context.Background(), cv)
}

func (s *Serializer) SerializeJSONCtx(ctx context.Context, cv *ComponentValue) ([]byte, error) {
	v, err := s.walkOutput(ctx, "", cv)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&v)
}

func (s *Serializer) walkOutput(ctx context.Context, breadcrumbs string, cv *ComponentValue) (out interface{}, err error) {
	if cv.Component == nil {
		return nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, cv)
	}
	switch cv.Component.ComponentType() {
	case ElementaryComponent:
		return s.serializeElementaryType(ctx, breadcrumbs, cv)
	case FixedArrayComponent, DynamicArrayComponent:
		return s.serializeArray(ctx, breadcrumbs, cv)
	case TupleComponent:
		return s.serializeTuple(ctx, breadcrumbs, cv)
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, cv.Component)
	}
}

func (s *Serializer) serializeElementaryType(ctx context.Context, breadcrumbs string, cv *ComponentValue) (interface{}, error) {
	switch cv.Component.ElementaryType() {
	case ElementaryTypeInt, ElementaryTypeUint:
		return s.is(cv.Value.(*big.Int)), nil
	case ElementaryTypeAddress:
		addr := make([]byte, 20)
		cv.Value.(*big.Int).FillBytes(addr)
		return s.bs(addr), nil
	case ElementaryTypeBool:
		return (cv.Value.(*big.Int).Int64() == 1), nil
	case ElementaryTypeFixed, ElementaryTypeUfixed:
		return s.fs(cv.Value.(*big.Float)), nil
	case ElementaryTypeBytes, ElementaryTypeFunction:
		return s.bs(cv.Value.([]byte)), nil
	case ElementaryTypeString:
		return cv.Value.(string), nil
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgUnknownABIElementaryType, cv.Component.ElementaryType(), breadcrumbs)
	}
}

func (s *Serializer) serializeArray(ctx context.Context, breadcrumbs string, cv *ComponentValue) (interface{}, error) {
	out := make([]interface{}, len(cv.Children))
	for i, child := range cv.Children {
		v, err := s.walkOutput(ctx, fmt.Sprintf("%s[%d]", breadcrumbs, i), child)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (s *Serializer) serializeTuple(ctx context.Context, breadcrumbs string, cv *ComponentValue) (interface{}, error) {
	switch s.ts {
	case FormatAsObjects:
		out := make(map[string]interface{})
		for i, child := range cv.Children {
			if child.Component != nil {
				name := child.Component.KeyName()
				if name == "" {
					name = s.dn(i)
				}
				v, err := s.walkOutput(ctx, fmt.Sprintf("%s[%s]", breadcrumbs, name), child)
				if err != nil {
					return nil, err
				}
				out[name] = v
			}
		}
		return out, nil
	case FormatAsFlatArrays:
		out := make([]interface{}, len(cv.Children))
		for i, child := range cv.Children {
			v, err := s.walkOutput(ctx, fmt.Sprintf("%s[%d]", breadcrumbs, i), child)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case FormatAsSelfDescribingArrays:
		out := make([]interface{}, len(cv.Children))
		for i, child := range cv.Children {
			vm := make(map[string]interface{})
			if child.Component != nil {
				vm["name"] = child.Component.KeyName()
				vm["type"] = child.Component.String()
			}
			if vm["name"] == "" {
				vm["name"] = s.dn(i)
			}
			v, err := s.walkOutput(ctx, fmt.Sprintf("%s[%s]", breadcrumbs, vm["name"]), child)
			if err != nil {
				return nil, err
			}
			vm["value"] = v
			out[i] = vm
		}
		return out, nil
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgUnknownTupleSerializer, s.ts)
	}
}
