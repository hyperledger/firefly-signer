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
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

var (
	int64Type    = reflect.TypeOf(int64(0))
	float64Type  = reflect.TypeOf(float64(0))
	stringerType = reflect.TypeOf(new(fmt.Stringer)).Elem()
)

// ComponentValue is a the ABI matched data associated with the TypeComponent tree of the ABI.
type ComponentValue struct {
	Component TypeComponent
	Leaf      bool
	Children  []*ComponentValue
	Value     interface{}
}

// getPtrValOrRawTypeNil sees if v is a pointer, with a non-nil value. If so returns that value, else nil
func getPtrValOrNil(v interface{}) interface{} {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		return val.Elem().Interface()
	}
	return nil
}

// getStringIfConvertible returns a string if it can be converted, and a bool with a result
func getStringIfConvertible(v interface{}) (string, bool) {
	vt := reflect.TypeOf(v)
	if vt == nil {
		return "", false
	}
	// We do a kind check here, rather than convertible check, because almost all low level types
	// are convertible to string - but you get a horrible results for integers etc.
	if vt.Kind() == reflect.String {
		return reflect.ValueOf(v).String(), true
	}
	if vt.Implements(stringerType) {
		return v.(fmt.Stringer).String(), true
	}
	return "", false
}

// getBytesIfConvertible returns a byte array if the type has that kind
func getBytesIfConvertible(v interface{}) []byte {
	vt := reflect.TypeOf(v)
	if vt == nil {
		return nil
	}
	if vt.Kind() == reflect.Slice && vt.Elem().Kind() == reflect.Uint8 {
		return reflect.ValueOf(v).Bytes()
	}
	return nil
}

// getInt64IfConvertible returns an int64 if it can be converted, and a bool with a result
func getInt64IfConvertible(v interface{}) (int64, bool) {
	vt := reflect.TypeOf(v)
	if vt == nil {
		return 0, false
	}
	if vt.ConvertibleTo(int64Type) {
		return reflect.ValueOf(v).Convert(int64Type).Interface().(int64), true
	}
	return 0, false
}

// getFloat64IfConvertible returns a float64 if it can be converted, and a bool with a result
func getFloat64IfConvertible(v interface{}) (float64, bool) {
	vt := reflect.TypeOf(v)
	if vt == nil {
		return 0, false
	}
	if vt.ConvertibleTo(float64Type) {
		return reflect.ValueOf(v).Convert(float64Type).Interface().(float64), true
	}
	return 0, false
}

// getIntegerFromInterface takes a bunch of types that could be passed in via Go,
// with a focus on those generated by the result of an Unmarshal using Go's default
// unmarshalling.
func getIntegerFromInterface(ctx context.Context, desc string, v interface{}) (*big.Int, error) {
	i := new(big.Int)
	switch vt := v.(type) {
	case string:
		// We use Go's default '0' base integer parsing, where `0x` means hex,
		// no prefix means decimal etc.
		i, ok := i.SetString(vt, 0)
		if !ok {
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidIntegerABIInput, vt, v, desc)
		}
		return i, nil
	case *big.Float:
		i, _ := vt.Int(i)
		return i, nil
	case *big.Int:
		return vt, nil
	case float64:
		// This is how JSON numbers come in (no distinction between integers/floats)
		i.SetInt64(int64(vt))
		return i, nil
	case float32:
		i.SetInt64(int64(vt))
		return i, nil
	case int64:
		i.SetInt64(vt)
		return i, nil
	case int32:
		i.SetInt64(int64(vt))
		return i, nil
	case int16:
		i.SetInt64(int64(vt))
		return i, nil
	case int8:
		i.SetInt64(int64(vt))
		return i, nil
	case int:
		i.SetInt64(int64(vt))
		return i, nil
	case uint64:
		i.SetInt64(int64(vt))
		return i, nil
	case uint32:
		i.SetInt64(int64(vt))
		return i, nil
	case uint16:
		i.SetInt64(int64(vt))
		return i, nil
	case uint8:
		i.SetInt64(int64(vt))
		return i, nil
	case uint:
		i.SetInt64(int64(vt))
		return i, nil
	default:
		if str, ok := getStringIfConvertible(v); ok {
			return getIntegerFromInterface(ctx, desc, str)
		}
		vi := getPtrValOrNil(v)
		if vi != nil {
			return getIntegerFromInterface(ctx, desc, vi)
		}
		if i64, ok := getInt64IfConvertible(v); ok {
			return getIntegerFromInterface(ctx, desc, i64)
		}
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidIntegerABIInput, vt, v, desc)
	}
}

// getFloatFromInterface takes a bunch of types that could be passed in via Go,
// with a focus on those generated by the result of an Unmarshal using Go's default
// unmarshalling.
func getFloatFromInterface(ctx context.Context, desc string, v interface{}) (*big.Float, error) {
	f := new(big.Float)
	switch vt := v.(type) {
	case string:
		// We use Go's default '0' base float parsing, where `0x` means hex,
		// no prefix means decimal etc.
		f, _, err := f.Parse(vt, 0)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidFloatABIInput, vt, v, desc)
		}
		return f, nil
	case *big.Float:
		return vt, nil
	case *big.Int:
		return f.SetInt(vt), nil
	case float64:
		// This is how JSON numbers come in (no distinction between integers/floats)
		f.SetFloat64(vt)
		return f, nil
	case float32:
		f.SetFloat64(float64(vt))
		return f, nil
	case int64:
		f.SetFloat64(float64(vt))
		return f, nil
	case int32:
		f.SetFloat64(float64(vt))
		return f, nil
	case int16:
		f.SetFloat64(float64(vt))
		return f, nil
	case int8:
		f.SetFloat64(float64(vt))
		return f, nil
	case int:
		f.SetFloat64(float64(vt))
		return f, nil
	case uint64:
		f.SetFloat64(float64(vt))
		return f, nil
	case uint32:
		f.SetFloat64(float64(vt))
		return f, nil
	case uint16:
		f.SetFloat64(float64(vt))
		return f, nil
	case uint8:
		f.SetFloat64(float64(vt))
		return f, nil
	case uint:
		f.SetFloat64(float64(vt))
		return f, nil
	default:
		if str, ok := getStringIfConvertible(v); ok {
			return getFloatFromInterface(ctx, desc, str)
		}
		vi := getPtrValOrNil(v)
		if vi != nil {
			return getFloatFromInterface(ctx, desc, vi)
		}
		if f64, ok := getFloat64IfConvertible(v); ok {
			return getFloatFromInterface(ctx, desc, f64)
		}
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidFloatABIInput, vt, v, desc)
	}
}

// getBoolFromInterface handles bool or string values - no attempt made to map
// integer types to bool
func getBoolFromInterface(ctx context.Context, desc string, v interface{}) (bool, error) {
	switch vt := v.(type) {
	case bool:
		return vt, nil
	case string:
		return strings.EqualFold(vt, "true"), nil
	default:
		if str, ok := getStringIfConvertible(v); ok {
			return getBoolFromInterface(ctx, desc, str)
		}
		vi := getPtrValOrNil(v)
		if vi != nil {
			return getBoolFromInterface(ctx, desc, vi)
		}
		return false, i18n.NewError(ctx, signermsgs.MsgInvalidBoolABIInput, vt, v, desc)
	}
}

// getStringFromInterface converts a go interface that is either a string,
// fmt.Stringable or []byte and returns the string value directly (without
// attempting hex decoding etc.)
func getStringFromInterface(ctx context.Context, desc string, v interface{}) (string, error) {
	switch vt := v.(type) {
	case string:
		return vt, nil
	case []byte:
		return string(vt), nil
	default:
		if str, ok := getStringIfConvertible(v); ok {
			return str, nil
		}
		vi := getPtrValOrNil(v)
		if vi != nil {
			return getStringFromInterface(ctx, desc, vi)
		}
		return "", i18n.NewError(ctx, signermsgs.MsgInvalidStringABIInput, vt, v, desc)
	}
}

// getBytesFromInterface converts input that can be either raw bytes in Go,
// or hex encoded (with or without 0x prefix) string data.
func getBytesFromInterface(ctx context.Context, desc string, v interface{}) ([]byte, error) {
	switch vt := v.(type) {
	case []byte:
		return vt, nil
	case string:
		vt = strings.TrimPrefix(vt, "0x")
		hb, err := hex.DecodeString(vt)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidHexABIInput, vt, v, desc)
		}
		return hb, nil
	default:
		if ba := getBytesIfConvertible(v); ba != nil {
			return ba, nil
		}
		if str, ok := getStringIfConvertible(v); ok {
			return getBytesFromInterface(ctx, desc, str)
		}
		vi := getPtrValOrNil(v)
		if vi != nil {
			return getBytesFromInterface(ctx, desc, vi)
		}
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidHexABIInput, vt, v)
	}
}

func getInterfaceArray(input interface{}) []interface{} {
	iArray, ok := input.([]interface{})
	if !ok {
		iv := reflect.ValueOf(input)
		iArray = make([]interface{}, iv.Len())
		for i := 0; i < iv.Len(); i++ {
			iArray[i] = iv.Index(i).Interface()
		}
	}
	return iArray
}

func getStringInterfaceMap(ctx context.Context, breadcrumbs string, input interface{}) (map[string]interface{}, error) {
	iMap, ok := input.(map[string]interface{})
	if !ok {
		iv := reflect.ValueOf(input)
		iMap = make(map[string]interface{}, iv.Len())
		iter := iv.MapRange()
		for iter.Next() {
			k, err := getStringFromInterface(ctx, breadcrumbs, iter.Key().Interface())
			if err != nil {
				return nil, err
			}
			iMap[k] = iter.Value().Interface()
		}
	}
	return iMap, nil
}

func walkInput(ctx context.Context, breadcrumbs string, input interface{}, component *typeComponent) (cv *ComponentValue, err error) {
	switch component.cType {
	case ElementaryComponent:
		value, err := component.elementaryType.readInput(ctx, breadcrumbs, input)
		if err != nil {
			return nil, err
		}
		return &ComponentValue{
			Component: component,
			Value:     value,
			Leaf:      true,
		}, nil
	case FixedArrayComponent, VariableArrayComponent:
		return walkArrayInput(ctx, breadcrumbs, input, component)
	case TupleComponent:
		return walkTupleInput(ctx, breadcrumbs, input, component)
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, component.cType)
	}

}

func walkArrayInput(ctx context.Context, breadcrumbs string, input interface{}, component *typeComponent) (cv *ComponentValue, err error) {
	vt := reflect.TypeOf(input)
	if vt == nil || vt.Kind() != reflect.Slice {
		return nil, i18n.NewError(ctx, signermsgs.MsgMustBeSliceABIInput, input, breadcrumbs)
	}
	iArray := getInterfaceArray(input)
	if component.cType == FixedArrayComponent && len(iArray) != component.arrayLength {
		return nil, i18n.NewError(ctx, signermsgs.MsgFixedLengthABIArrayMismatch, len(iArray), component.arrayLength, breadcrumbs)
	}
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, len(iArray)),
	}
	for i, v := range iArray {
		childBreadcrumbs := fmt.Sprintf("%s[%d]", breadcrumbs, i)
		cv.Children[i], err = walkInput(ctx, childBreadcrumbs, v, component.arrayChild)
		if err != nil {
			return nil, err
		}
	}
	return cv, nil
}

func walkTupleInputArray(ctx context.Context, breadcrumbs string, input interface{}, component *typeComponent) (cv *ComponentValue, err error) {
	iArray := getInterfaceArray(input)
	if len(iArray) != len(component.tupleChildren) {
		return nil, i18n.NewError(ctx, signermsgs.MsgTupleABIArrayMismatch, len(iArray), len(component.tupleChildren), breadcrumbs)
	}
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, len(iArray)),
	}
	for i, v := range iArray {
		childBreadcrumbs := fmt.Sprintf("%s.%d", breadcrumbs, i)
		cv.Children[i], err = walkInput(ctx, childBreadcrumbs, v, component.tupleChildren[i])
		if err != nil {
			return nil, err
		}
	}
	return cv, nil
}

func walkTupleInput(ctx context.Context, breadcrumbs string, input interface{}, component *typeComponent) (cv *ComponentValue, err error) {
	vt := reflect.TypeOf(input)
	if vt != nil && vt.Kind() == reflect.Slice {
		return walkTupleInputArray(ctx, breadcrumbs, input, component)
	}
	if vt == nil || vt.Kind() != reflect.Map {
		return nil, i18n.NewError(ctx, signermsgs.MsgTupleABINotArrayOrMap, input, breadcrumbs)
	}
	iMap, err := getStringInterfaceMap(ctx, breadcrumbs, input)
	if err != nil {
		return nil, err
	}
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, len(iMap)),
	}
	for i, tupleChild := range component.tupleChildren {
		if tupleChild.keyName == "" {
			return nil, i18n.NewError(ctx, signermsgs.MsgTupleInABINoName, i, breadcrumbs)
		}
		childBreadcrumbs := fmt.Sprintf("%s.%s", breadcrumbs, tupleChild.keyName)
		v, ok := iMap[tupleChild.keyName]
		if !ok {
			return nil, i18n.NewError(ctx, signermsgs.MsgMissingInputKeyABITuple, tupleChild.keyName, childBreadcrumbs)
		}
		cv.Children[i], err = walkInput(ctx, childBreadcrumbs, v, component.tupleChildren[i])
		if err != nil {
			return nil, err
		}
	}
	return cv, nil
}
