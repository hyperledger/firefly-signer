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
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStringCustomType string
type TestStringPtrCustomType *string
type TestInt64CustomType int64
type TestInt64PtrCustomType *int64
type TestInt32CustomType int32
type TestInt32PtrCustomType *int32
type TestFloat64CustomType float64
type TestFloat64PtrCustomType *float64
type TestFloat32CustomType float32
type TestFloat32PtrCustomType *float32
type TestByteArrayCustomType []byte
type TestByteArrayPtrCustomType *[]byte

type TestStringable struct {
	s string
}

func (ts *TestStringable) String() string {
	return ts.s
}

func TestGetIntegerFromInterface(t *testing.T) {

	ctx := context.Background()

	i, err := getIntegerFromInterface(ctx, "-12345")
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "4276993775", i.String())

	i, err = getIntegerFromInterface(ctx, "-0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "-4276993775", i.String())

	i, err = getIntegerFromInterface(ctx, int(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, int64(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, int32(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, int16(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, int8(-32))
	assert.NoError(t, err)
	assert.Equal(t, "-32", i.String())

	i, err = getIntegerFromInterface(ctx, uint(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, uint64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, uint32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, uint16(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, uint8(32))
	assert.NoError(t, err)
	assert.Equal(t, "32", i.String())

	i, err = getIntegerFromInterface(ctx, float64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, float32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	iB := big.NewInt(-12345)
	i, err = getIntegerFromInterface(ctx, iB)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	iF := big.NewFloat(-12345)
	i, err = getIntegerFromInterface(ctx, iF)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	str := "-12345"
	i, err = getIntegerFromInterface(ctx, &str)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	strPtr := &str
	i, err = getIntegerFromInterface(ctx, &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, &TestStringable{s: "-12345"})
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var is TestStringCustomType = "-12345"
	i, err = getIntegerFromInterface(ctx, &is)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var ips TestStringPtrCustomType = strPtr
	i, err = getIntegerFromInterface(ctx, ips)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI64 TestInt64CustomType = -12345
	i, err = getIntegerFromInterface(ctx, &iI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i64 := int64(-12345)
	var iPI64 TestInt64PtrCustomType = &i64
	i, err = getIntegerFromInterface(ctx, iPI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI32 TestInt32CustomType = -12345
	i, err = getIntegerFromInterface(ctx, &iI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i32 := int32(-12345)
	var iPI32 TestInt32PtrCustomType = &i32
	i, err = getIntegerFromInterface(ctx, iPI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "wrong")
	assert.Regexp(t, "FF00163", err)
	assert.Nil(t, i)

	i, err = getIntegerFromInterface(ctx, []string{"wrong"})
	assert.Regexp(t, "FF00163", err)
	assert.Nil(t, i)

}

func TestGetFloatFromInterface(t *testing.T) {

	ctx := context.Background()

	i, err := getFloatFromInterface(ctx, "-1.2345")
	assert.NoError(t, err)
	assert.Equal(t, "-1.2345", i.String())

	i, err = getFloatFromInterface(ctx, "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "4276993775", i.String())

	i, err = getFloatFromInterface(ctx, "-0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "-4276993775", i.String())

	i, err = getFloatFromInterface(ctx, int(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, int64(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, int32(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, int16(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, int8(-32))
	assert.NoError(t, err)
	assert.Equal(t, "-32", i.String())

	i, err = getFloatFromInterface(ctx, uint(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, uint64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, uint32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, uint16(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, uint8(32))
	assert.NoError(t, err)
	assert.Equal(t, "32", i.String())

	i, err = getFloatFromInterface(ctx, float64(-1.2345))
	assert.NoError(t, err)
	assert.Equal(t, "-1.2345", i.String())

	i, err = getFloatFromInterface(ctx, float32(1.2345))
	assert.NoError(t, err)
	assert.Equal(t, "1.234500051", i.String())

	iB := big.NewInt(-12345)
	i, err = getFloatFromInterface(ctx, iB)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	iF := big.NewFloat(-12345)
	i, err = getFloatFromInterface(ctx, iF)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	str := "-12345"
	i, err = getFloatFromInterface(ctx, &str)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	strPtr := &str
	i, err = getFloatFromInterface(ctx, &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var is TestStringCustomType = "-12345"
	i, err = getFloatFromInterface(ctx, &is)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var ips TestStringPtrCustomType = strPtr
	i, err = getFloatFromInterface(ctx, ips)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI64 TestInt64CustomType = -12345
	i, err = getFloatFromInterface(ctx, &iI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i64 := int64(-12345)
	var iPI64 TestInt64PtrCustomType = &i64
	i, err = getFloatFromInterface(ctx, iPI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI32 TestInt32CustomType = -12345
	i, err = getFloatFromInterface(ctx, &iI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i32 := int32(-12345)
	var iPI32 TestInt32PtrCustomType = &i32
	i, err = getFloatFromInterface(ctx, iPI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "wrong")
	assert.Regexp(t, "FF00164", err)
	assert.Nil(t, i)

	i, err = getFloatFromInterface(ctx, []string{"wrong"})
	assert.Regexp(t, "FF00164", err)
	assert.Nil(t, i)

}

func TestGetBoolFromInterface(t *testing.T) {

	ctx := context.Background()

	v, err := getBoolFromInterface(ctx, "true")
	assert.NoError(t, err)
	assert.True(t, v)

	v, err = getBoolFromInterface(ctx, "false")
	assert.NoError(t, err)
	assert.False(t, v)

	v, err = getBoolFromInterface(ctx, true)
	assert.NoError(t, err)
	assert.True(t, v)

	v, err = getBoolFromInterface(ctx, false)
	assert.NoError(t, err)
	assert.False(t, v)

	vTrue := true
	v, err = getBoolFromInterface(ctx, &vTrue)
	assert.NoError(t, err)
	assert.True(t, v)

	var is TestStringCustomType = "true"
	v, err = getBoolFromInterface(ctx, &is)
	assert.NoError(t, err)
	assert.True(t, v)

	_, err = getBoolFromInterface(ctx, int(-12345))
	assert.Regexp(t, "FF00166", err)

	_, err = getBoolFromInterface(ctx, []bool{true})
	assert.Regexp(t, "FF00166", err)

}

func TestGetStringFromInterface(t *testing.T) {

	ctx := context.Background()

	s, err := getStringFromInterface(ctx, "test data")
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	s, err = getStringFromInterface(ctx, "test data")
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	s, err = getStringFromInterface(ctx, []byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	str := "test data"
	s, err = getStringFromInterface(ctx, &str)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	strPtr := &str
	s, err = getStringFromInterface(ctx, &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	var is TestStringCustomType = "test data"
	s, err = getStringFromInterface(ctx, &is)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	var ips TestStringPtrCustomType = strPtr
	s, err = getStringFromInterface(ctx, ips)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	_, err = getStringFromInterface(ctx, int(-12345))
	assert.Regexp(t, "FF00165", err)

	_, err = getStringFromInterface(ctx, []string{"wrong"})
	assert.Regexp(t, "FF00165", err)

}

func TestGetBytesFromInterface(t *testing.T) {

	ctx := context.Background()

	s, err := getBytesFromInterface(ctx, "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, "feedbeef")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, []byte{0xfe, 0xed, 0xbe, 0xef})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, []byte{0xfe, 0xed, 0xbe, 0xef})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	bv := []byte{0xfe, 0xed, 0xbe, 0xef}
	s, err = getBytesFromInterface(ctx, &bv)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var bt TestByteArrayCustomType = []byte{0xfe, 0xed, 0xbe, 0xef}
	s, err = getBytesFromInterface(ctx, bt)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var bpt TestByteArrayPtrCustomType = &bv
	s, err = getBytesFromInterface(ctx, bpt)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var is TestStringCustomType = "0xfeedbeef"
	s, err = getBytesFromInterface(ctx, &is)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	_, err = getBytesFromInterface(ctx, int(-12345))
	assert.Regexp(t, "FF00167", err)

	_, err = getBytesFromInterface(ctx, []string{"wrong"})
	assert.Regexp(t, "FF00167", err)

	_, err = getBytesFromInterface(ctx, "wrong")
	assert.Regexp(t, "FF00167", err)

}
