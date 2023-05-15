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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExampleABIDecode1(t *testing.T) {

	f := &Entry{
		Name: "baz",
		Inputs: ParameterArray{
			{Type: "uint32"},
			{Type: "bool"},
		},
	}

	d, err := hex.DecodeString("cdcd77c0" +
		"0000000000000000000000000000000000000000000000000000000000000045" +
		"0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)

	cv, err := f.DecodeCallData(d)
	assert.NoError(t, err)

	assert.Equal(t, "69", cv.Children[0].Value.(*big.Int).String())
	assert.Equal(t, "1", cv.Children[1].Value.(*big.Int).String()) // bool true

}

func TestExampleABIDecode2(t *testing.T) {

	f := &Entry{
		Name: "bar",
		Inputs: ParameterArray{
			{Type: "bytes3[2]"},
		},
	}

	d, err := hex.DecodeString("fce353f6" +
		"6162630000000000000000000000000000000000000000000000000000000000" + // first fixed-length entry in static array
		"6465660000000000000000000000000000000000000000000000000000000000", // second fixed-length in static array
	)
	assert.NoError(t, err)

	cv, err := f.DecodeCallData(d)
	assert.NoError(t, err)

	assert.Equal(t, "abc", string(cv.Children[0].Children[0].Value.([]byte)))
	assert.Equal(t, "def", string(cv.Children[0].Children[1].Value.([]byte)))

}

func TestExampleABIDecode3(t *testing.T) {

	f := &Entry{
		Name: "sam",
		Inputs: ParameterArray{
			{Type: "bytes"},
			{Type: "bool"},
			{Type: "uint[]"},
		},
	}

	d, err := hex.DecodeString("a5643bf2" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000060" + //   0: location of data for 1st param
		"0000000000000000000000000000000000000000000000000000000000000001" + //  32: boolean true
		"00000000000000000000000000000000000000000000000000000000000000a0" + //  64: location of data for 3rd param
		// 1st param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000004" + //  96: length in bytes
		"6461766500000000000000000000000000000000000000000000000000000000" + // 128: "dave" padded right
		// 3rd param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000003" + // 160: length of array
		"0000000000000000000000000000000000000000000000000000000000000001" + // 192: 1first value
		"0000000000000000000000000000000000000000000000000000000000000002" + // 224: second value
		"0000000000000000000000000000000000000000000000000000000000000003", //  256: third value
	)
	assert.NoError(t, err)

	cv, err := f.DecodeCallData(d)
	assert.NoError(t, err)

	assert.Equal(t, "dave", string(cv.Children[0].Value.([]byte)))
	assert.Equal(t, "1", cv.Children[1].Value.(*big.Int).String())
	assert.Len(t, cv.Children[2].Children, 3)
	assert.Equal(t, "1", cv.Children[2].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "2", cv.Children[2].Children[1].Value.(*big.Int).String())
	assert.Equal(t, "3", cv.Children[2].Children[2].Value.(*big.Int).String())

}

func TestExampleABIDecode4(t *testing.T) {

	f := &Entry{
		Name: "f",
		Inputs: ParameterArray{
			{Type: "uint"},
			{Type: "uint32[]"},
			{Type: "bytes10"},
			{Type: "bytes"},
		},
	}

	d, err := hex.DecodeString("8be65246" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000123" + // 0x123 padded to 32 b
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset of start of 2nd param
		"3132333435363738393000000000000000000000000000000000000000000000" + // 0x1234567890 padded to 32 b
		"00000000000000000000000000000000000000000000000000000000000000e0" + // offset of start of 4th param
		// 2nd param (dynamic)
		"0000000000000000000000000000000000000000000000000000000000000002" + // 2 elements in array
		"0000000000000000000000000000000000000000000000000000000000000456" + // first element
		"0000000000000000000000000000000000000000000000000000000000000789" + // second element
		// 4th param (dynamic)
		"000000000000000000000000000000000000000000000000000000000000000d" + // 13 bytes
		"48656c6c6f2c20776f726c642100000000000000000000000000000000000000", // the string
	)
	assert.NoError(t, err)

	cv, err := f.DecodeCallData(d)
	assert.NoError(t, err)

	assert.Equal(t, int64(0x123), cv.Children[0].Value.(*big.Int).Int64())
	assert.Len(t, cv.Children[1].Children, 2)
	assert.Equal(t, int64(0x456), cv.Children[1].Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(0x789), cv.Children[1].Children[1].Value.(*big.Int).Int64())
	assert.Equal(t, "1234567890", string(cv.Children[2].Value.([]byte)))
	assert.Equal(t, "Hello, world!", string(cv.Children[3].Value.([]byte)))

}

func TestExampleABIDecode5(t *testing.T) {

	f := &Entry{
		Name: "g",
		Inputs: ParameterArray{
			{Type: "uint[][]"},
			{Type: "string[]"},
		},
	}

	d, err := hex.DecodeString("2289b18c" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of [[1, 2], [3]]
		"0000000000000000000000000000000000000000000000000000000000000140" + // offset of ["one", "two", "three"]
		"0000000000000000000000000000000000000000000000000000000000000002" + // count for [[1, 2], [3]]
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of [1, 2]
		"00000000000000000000000000000000000000000000000000000000000000a0" + // offset of [3]
		"0000000000000000000000000000000000000000000000000000000000000002" + // count for [1, 2]
		"0000000000000000000000000000000000000000000000000000000000000001" + // encoding of 1
		"0000000000000000000000000000000000000000000000000000000000000002" + // encoding of 2
		"0000000000000000000000000000000000000000000000000000000000000001" + // count for [3]
		"0000000000000000000000000000000000000000000000000000000000000003" + // encoding of 3
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for ["one", "two", "three"]
		"0000000000000000000000000000000000000000000000000000000000000060" + // offset for "one"
		"00000000000000000000000000000000000000000000000000000000000000a0" + // offset for "two"
		"00000000000000000000000000000000000000000000000000000000000000e0" + // offset for "three"
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for "one"
		"6f6e650000000000000000000000000000000000000000000000000000000000" + // encoding of "one"
		"0000000000000000000000000000000000000000000000000000000000000003" + // count for "two"
		"74776f0000000000000000000000000000000000000000000000000000000000" + // encoding of "two"
		"0000000000000000000000000000000000000000000000000000000000000005" + // count for "three"
		"7468726565000000000000000000000000000000000000000000000000000000", // encoding of "three"
	)
	assert.NoError(t, err)

	cv, err := f.DecodeCallData(d)
	assert.NoError(t, err)

	assert.Len(t, cv.Children[0].Children, 2)
	assert.Len(t, cv.Children[0].Children[0].Children, 2)
	assert.Equal(t, int64(1), cv.Children[0].Children[0].Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(2), cv.Children[0].Children[0].Children[1].Value.(*big.Int).Int64())
	assert.Len(t, cv.Children[1].Children, 3)
	assert.Equal(t, "one", cv.Children[1].Children[0].Value)
	assert.Equal(t, "two", cv.Children[1].Children[1].Value)
	assert.Equal(t, "three", cv.Children[1].Children[2].Value)

}

func TestExampleABIDecode6(t *testing.T) {

	params := &ParameterArray{
		{Type: "uint256[3]"},
		{Type: "address"},
		{Type: "string[2]"},
		{Type: "bool"},
	}

	bignumber, _ := big.NewInt(0).SetString("30000000000000000000", 10)
	values := []interface{}{
		[]*big.Int{big.NewInt(1545304298), big.NewInt(6), bignumber},
		"ab1257528b3782fb40d7ed5f72e624b744dffb2f",
		[]string{"Ethereum", "Hello, Ethereum!"},
		false,
	}

	enc, err := params.EncodeABIDataValues(values)
	assert.NoError(t, err)

	assert.Equal(t,
		"000000000000000000000000000000000000000000000000000000005c1b78ea"+ //       0: 1545304298           inline in head
			"0000000000000000000000000000000000000000000000000000000000000006"+ //  32: 6                    inline in head
			"000000000000000000000000000000000000000000000001a055690d9db80000"+ //  64: 30000000000000000000 inline in head
			"000000000000000000000000ab1257528b3782fb40d7ed5f72e624b744dffb2f"+ //  96: address              inline in head
			"00000000000000000000000000000000000000000000000000000000000000c0"+ // 128: offset of string[2] = 192
			"0000000000000000000000000000000000000000000000000000000000000000"+ // 160: bool - false         inline in head
			"0000000000000000000000000000000000000000000000000000000000000040"+ // 192: offset of first string = 64 (+192 = 256)
			"0000000000000000000000000000000000000000000000000000000000000080"+ // 224: offset of second string = 128 (+192 = 320)
			"0000000000000000000000000000000000000000000000000000000000000008"+ // 256: string length = 8
			"457468657265756d000000000000000000000000000000000000000000000000"+ // 288: "Ethereum"
			"0000000000000000000000000000000000000000000000000000000000000010"+ // 320: string length = 16
			"48656c6c6f2c20457468657265756d2100000000000000000000000000000000", // 352: "Hello, Ethereum!"
		hex.EncodeToString(enc))

	cv, err := params.DecodeABIData(enc, 0)
	assert.NoError(t, err)

	assert.Equal(t, big.NewInt(1545304298), cv.Children[0].Children[0].Value)
	assert.Equal(t, big.NewInt(6), cv.Children[0].Children[1].Value)
	assert.Equal(t, bignumber, cv.Children[0].Children[2].Value)
	address, _ := cv.Children[1].JSON()
	assert.Equal(t, "\"ab1257528b3782fb40d7ed5f72e624b744dffb2f\"", string(address))
	assert.Equal(t, "Ethereum", cv.Children[2].Children[0].Value)
	assert.Equal(t, "Hello, Ethereum!", cv.Children[2].Children[1].Value)
	boolean, _ := cv.Children[3].JSON()
	assert.Equal(t, "false", string(boolean))
}

func TestExampleABIDecode7(t *testing.T) {

	// a tuple of dynamic types (which is the same as a fixed-length array of the dynamic types)
	f := &Entry{
		Name: "g",
		Outputs: ParameterArray{
			{
				Type: "tuple",
				Components: ParameterArray{
					{Type: "string"},
					{Type: "string"},
				},
			},
		},
	}

	d, _ := hex.DecodeString("" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset of ["c", "d"]
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of "c"
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset of "d"
		"0000000000000000000000000000000000000000000000000000000000000001" + // count of "c"
		"6300000000000000000000000000000000000000000000000000000000000000" + // encoding of "c"
		"0000000000000000000000000000000000000000000000000000000000000001" + // count of "d"
		"6400000000000000000000000000000000000000000000000000000000000000", // encoding of "d"
	)

	cv, err := f.Outputs.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, "c", cv.Children[0].Children[0].Value)
	assert.Equal(t, "d", cv.Children[0].Children[1].Value)
}

func TestExampleABIDecodeTupleDifferentOrder(t *testing.T) {

	f := &Entry{
		Name: "coupon",
		Outputs: ParameterArray{
			{
				Type: "tuple[]",
				Name: "tokenspec",
				Components: ParameterArray{
					{Type: "uint256", Name: "_tokenId"},
					{Type: "uint256", Name: "_startDate"},
					{Type: "string", Name: "_tokenURL"},
					{Type: "uint256", Name: "_endDate"},
				},
			},
		},
	}

	value := "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000031079d0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000096e000000000000000000000000000000000000000000000000000000000000000b7468697369736d7975726c000000000000000000000000000000000000000000"

	d, _ := hex.DecodeString(value)

	cv, err := f.Outputs.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, big.NewInt(123), cv.Children[0].Children[0].Children[0].Value)
	assert.Equal(t, big.NewInt(3213213), cv.Children[0].Children[0].Children[1].Value)
	assert.Equal(t, "thisismyurl", cv.Children[0].Children[0].Children[2].Value)
	assert.Equal(t, big.NewInt(2414), cv.Children[0].Children[0].Children[3].Value)
}

func TestExampleABIDecodeTuple(t *testing.T) {

	f := &Entry{
		Name: "coupon",
		Outputs: ParameterArray{
			{
				Type: "tuple[]",
				Name: "tokenspec",
				Components: ParameterArray{
					{Type: "uint256", Name: "_tokenId"},
					{Type: "string", Name: "_tokenURL"},
					{Type: "uint256", Name: "_startDate"},
					{Type: "uint256", Name: "_endDate"},
				},
			},
		},
	}

	d, _ := hex.DecodeString("" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000001" +
		"0000000000000000000000000000000000000000000000000000000000000020" + "000000000000000000000000000000000000000000000000000000000000007b" + "0000000000000000000000000000000000000000000000000000000000000080" + "000000000000000000000000000000000000000000000000000000000031079d" + "000000000000000000000000000000000000000000000000000000000000096e" + "000000000000000000000000000000000000000000000000000000000000000b" +
		"7468697369736d7975726c000000000000000000000000000000000000000000")

	cv, err := f.Outputs.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, big.NewInt(123), cv.Children[0].Children[0].Children[0].Value)
	assert.Equal(t, "thisismyurl", cv.Children[0].Children[0].Children[1].Value)
	assert.Equal(t, big.NewInt(3213213), cv.Children[0].Children[0].Children[2].Value)
	assert.Equal(t, big.NewInt(2414), cv.Children[0].Children[0].Children[3].Value)
}

func TestExampleABIDecodeDoubleNestedTuple(t *testing.T) {

	f := &Entry{
		Name: "f",
		Outputs: ParameterArray{
			{
				Type: "tuple[]",
				Name: "nested",
				Components: ParameterArray{
					{Type: "uint256", Name: "a"},
					{Type: "string", Name: "b"},
					{Type: "tuple", Name: "c", Components: ParameterArray{
						{Type: "uint256", Name: "c1"},
						{Type: "string", Name: "c2"},
						{Type: "uint256", Name: "c3"},
					}},
					{Type: "uint256", Name: "d"},
				},
			},
		},
	}

	b, err := f.Outputs.EncodeABIDataJSON([]byte(`{
		"nested": [{
			"a": 11111,
			"b": "test22222",
			"c": {
				"c1": 33333,
				"c2": "test44444",
				"c3": 55555
			},
			"d": 66666
		}]
	}`))
	assert.NoError(t, err)

	// The encoding looks right to me
	assert.Equal(t,
		"0000000000000000000000000000000000000000000000000000000000000020"+ // 0   - 32 - offset for the start of the dynamic array
			"0000000000000000000000000000000000000000000000000000000000000001"+ // 32  - 1  - number of tuples in the dynamic array
			"0000000000000000000000000000000000000000000000000000000000000020"+ // 64  - 32 - offset of the data for the tuple at position 0 in the array
			"0000000000000000000000000000000000000000000000000000000000002b67"+ // 96  - 11111 - value "a"
			"0000000000000000000000000000000000000000000000000000000000000080"+ // 128 - 128 - offset of the string data for "b", relative to 96 = 224
			"00000000000000000000000000000000000000000000000000000000000000c0"+ // 160 - 192 - offset of the tuple data for "c", relative to 96 = 288
			"000000000000000000000000000000000000000000000000000000000001046a"+ // 192 - 66666 - value of "d"
			"0000000000000000000000000000000000000000000000000000000000000009"+ // 224 - 9  - length of the string data for "b"
			"7465737432323232320000000000000000000000000000000000000000000000"+ // 256 - "test22222"
			"0000000000000000000000000000000000000000000000000000000000008235"+ // 288 - 33333 - value of "c1"
			"0000000000000000000000000000000000000000000000000000000000000060"+ // 320 - 96 - offset of the string data for "c2", relative to 288 = 384
			"000000000000000000000000000000000000000000000000000000000000d903"+ // 352 - 55555 - value of "c3"
			"0000000000000000000000000000000000000000000000000000000000000009"+ // 384 - 9 - length of the string data for "c2"
			"7465737434343434340000000000000000000000000000000000000000000000", // 416 - "test44444"
		hex.EncodeToString(b))

	cv, err := f.Outputs.DecodeABIData(b, 0)

	assert.Equal(t, big.NewInt(11111), cv.Children[0].Children[0].Children[0].Value)
	assert.Equal(t, "test22222", cv.Children[0].Children[0].Children[1].Value)
	assert.Equal(t, big.NewInt(33333), cv.Children[0].Children[0].Children[2].Children[0].Value)
	assert.Equal(t, "test44444", cv.Children[0].Children[0].Children[2].Children[1].Value)
	assert.Equal(t, big.NewInt(55555), cv.Children[0].Children[0].Children[2].Children[2].Value)
	assert.Equal(t, big.NewInt(66666), cv.Children[0].Children[0].Children[3].Value)
}

func TestExampleABIDecodeTupleFixed(t *testing.T) {

	f := &Entry{
		Name: "coupon",
		Outputs: ParameterArray{
			{
				Type: "tuple[1]",
				Components: ParameterArray{
					{Type: "string", Name: "value"},
				},
			},
		},
	}

	d, _ := hex.DecodeString("" +
		"0000000000000000000000000000000000000000000000000000000000000020" + "0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000020" + "0000000000000000000000000000000000000000000000000000000000000008" + "6d79737472696e67000000000000000000000000000000000000000000000000")

	cv, err := f.Outputs.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, "mystring", cv.Children[0].Children[0].Children[0].Value)
}

func TestExampleABIDecode8(t *testing.T) {

	// a fixed-length array of dynamic types
	f := &Entry{
		Name: "g",
		Outputs: ParameterArray{
			{Type: "string[2]"},
		},
	}

	d, _ := hex.DecodeString("" +
		// head
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset of ["c", "d"]
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset of "c"
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset of "d"
		"0000000000000000000000000000000000000000000000000000000000000001" + // count of "c"
		"6300000000000000000000000000000000000000000000000000000000000000" + // encoding of "c"
		"0000000000000000000000000000000000000000000000000000000000000001" + // count of "d"
		"6400000000000000000000000000000000000000000000000000000000000000", // encoding of "d"
	)

	cv, err := f.Outputs.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, "c", cv.Children[0].Children[0].Value)
	assert.Equal(t, "d", cv.Children[0].Children[1].Value)
}

func TestDecodeABISignedIntOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "int"},
	}
	d, err := hex.DecodeString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffedcbb")
	assert.NoError(t, err)

	tc, err := p.TypeComponentTree()
	assert.NoError(t, err)
	cv, err := tc.DecodeABIData(d, 0)
	assert.NoError(t, err)

	assert.Equal(t, ElementaryTypeInt, cv.Children[0].Component.ElementaryType())
	assert.Equal(t, int64(-0x12345), cv.Children[0].Value.(*big.Int).Int64())

}

func TestDecodeABISignedIntTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "int"},
	}
	d, err := hex.DecodeString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffedc")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIUnsignedIntOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000012345")
	assert.NoError(t, err)

	cv, err := p.DecodeABIData(d, 0)
	assert.NoError(t, err)
	assert.Equal(t, ElementaryTypeUint, cv.Children[0].Component.ElementaryType())
	assert.Equal(t, int64(0x12345), cv.Children[0].Value.(*big.Int).Int64())

}

func TestDecodeABIUnsignedIntTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint"},
	}
	d, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000123")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIFixedOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "fixed64x4"},
	}
	d, err := hex.DecodeString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffedcbb")
	assert.NoError(t, err)

	cv, err := p.DecodeABIData(d, 0)
	assert.NoError(t, err)
	assert.Equal(t, ElementaryTypeFixed, cv.Children[0].Component.ElementaryType())
	assert.Equal(t, "-7.4565", cv.Children[0].Value.(*big.Float).String())

}

func TestDecodeABIFixedTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "fixed64x4"},
	}
	d, err := hex.DecodeString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffedc")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIUfixedOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "ufixed64x4"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000012345")
	assert.NoError(t, err)

	cv, err := p.DecodeABIData(d, 0)
	assert.NoError(t, err)
	assert.Equal(t, ElementaryTypeUfixed, cv.Children[0].Component.ElementaryType())
	assert.Equal(t, "7.4565", cv.Children[0].Value.(*big.Float).String())

}

func TestDecodeABIUfixedTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "ufixed64x4"},
	}
	d, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000123")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestIntToFixedBadComponent(t *testing.T) {
	_, err := intToFixed(context.Background(), &typeComponent{cType: TupleComponent}, &ComponentValue{})
	assert.Regexp(t, "FF22041", err)
}

func TestDecodeABIDynamicArrayTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint256[]"},
	}
	d, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000123")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22045", err)

}

func TestDecodeABIDynamicArrayTooLong(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint256[]"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020" +
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	assert.NoError(t, err)

	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22046", err)

}

func TestDecodeABIBytesFixedOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "bytes13"},
	}
	d, err := hex.DecodeString("48656c6c6f2c20776f726c642100000000000000000000000000000000000000")
	assert.NoError(t, err)

	cv, err := p.DecodeABIData(d, 0)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", string(cv.Children[0].Value.([]byte)))

}

func TestDecodeABIStringVariableOk(t *testing.T) {

	p := &ParameterArray{
		{Type: "string"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020" +
		"000000000000000000000000000000000000000000000000000000000000000d" +
		"48656c6c6f2c20776f726c642100000000000000000000000000000000000000")
	assert.NoError(t, err)

	cv, err := p.DecodeABIData(d, 0)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", cv.Children[0].Value)

}

func TestDecodeABIStringVariableInsufficientBytesForLength(t *testing.T) {

	p := &ParameterArray{
		{Type: "string"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020" +
		"00000000000000000000000000000000000000000000000000000000000000")
	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22045", err)

}

func TestDecodeABIStringVariableInsufficientBytesForOffset(t *testing.T) {

	p := &ParameterArray{
		{Type: "string"},
	}
	d, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000")
	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22045", err)

}

func TestDecodeABIStringVariableInsufficientBytesForValue(t *testing.T) {

	p := &ParameterArray{
		{Type: "string"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020" +
		"00000000000000000000000000000000000000000000000000000000000000ff")
	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIFixedArrayTooFewBytes(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint256[1]"},
	}
	d, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000")
	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIDynamicArrayTooFewBytesForValue(t *testing.T) {

	p := &ParameterArray{
		{Type: "uint256[]"},
	}
	d, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000020")
	_, err = p.DecodeABIData(d, 0)
	assert.Regexp(t, "FF22047", err)

}

func TestDecodeABIElementBadComponent(t *testing.T) {
	_, _, err := decodeABIElement(context.Background(), "", []byte{}, 0, 0, &typeComponent{
		cType: 99,
	})
	assert.Regexp(t, "FF22041", err)
}

func TestDecodeABIDataBadParam(t *testing.T) {

	p := &ParameterArray{
		{Type: "wrong"},
	}

	_, err := p.DecodeABIData([]byte{}, 0)
	assert.Regexp(t, "FF22025", err)

}

func TestDecodeCallDataInsufficientSigBytes(t *testing.T) {

	f := &Entry{
		Name:   "doit",
		Inputs: ParameterArray{},
	}

	d, err := hex.DecodeString("ffffff")
	assert.NoError(t, err)
	_, err = f.DecodeCallData(d)
	assert.Regexp(t, "FF22048", err)
}

func TestDecodeCallDataWrongSigBytes(t *testing.T) {

	f := &Entry{
		Name:   "doit",
		Inputs: ParameterArray{},
	}

	d, err := hex.DecodeString("ffffffff")
	assert.NoError(t, err)
	_, err = f.DecodeCallData(d)
	assert.Regexp(t, "FF22049", err)
}

func TestDecodeCallDataSigGenerationFailed(t *testing.T) {

	f := &Entry{
		Name: "doit",
		Inputs: ParameterArray{
			{Type: "wrong"},
		},
	}

	d, err := hex.DecodeString("ffffffff")
	assert.NoError(t, err)
	_, err = f.DecodeCallData(d)
	assert.Regexp(t, "FF22025", err)
}

func TestIsDynamicType(t *testing.T) {
	p := &ParameterArray{
		{Type: "int"},
		{Type: "int[]"},
		{Type: "string[0]"},
		{Type: "string[1]"},
		{Type: "tuple", Components: ParameterArray{}},
		{Type: "tuple", Components: ParameterArray{
			{Type: "int"},
			{Type: "bytes3"},
		}},
		{Type: "tuple", Components: ParameterArray{
			{Type: "int"},
			{Type: "string"},
		}},
		{Type: "tuple", Components: ParameterArray{
			{Type: "int"},
			{Type: "int[]"},
		}},
	}
	tc, err := p.TypeComponentTree()
	assert.NoError(t, err)
	ctx := context.Background()

	// Bad type
	_, err = isDynamicType(ctx, &typeComponent{cType: 99})
	assert.Regexp(t, "FF22041", err)

	// Fixed type
	dt, err := isDynamicType(ctx, tc.(*typeComponent).tupleChildren[0])
	assert.NoError(t, err)
	assert.False(t, dt)

	// Dynamic array of fixed type
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[1])
	assert.NoError(t, err)
	assert.True(t, dt)

	// Zero length fixed array of dynamic length type
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[2])
	assert.NoError(t, err)
	assert.False(t, dt)

	// Non-zero length fixed array of dynamic length type
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[3])
	assert.NoError(t, err)
	assert.True(t, dt)

	// Zero length tuple
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[4])
	assert.NoError(t, err)
	assert.False(t, dt)

	// Non-zero length tuple with fixed types
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[5])
	assert.NoError(t, err)
	assert.False(t, dt)

	// Non-zero length tuple with simple dynamic types
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[6])
	assert.NoError(t, err)
	assert.True(t, dt)

	// Non-zero length tuple with simple dynamic array type
	dt, err = isDynamicType(ctx, tc.(*typeComponent).tupleChildren[7])
	assert.NoError(t, err)
	assert.True(t, dt)
}

func TestIsDynamicTypeBadNestedTupleType(t *testing.T) {
	_, err := isDynamicType(context.Background(), &typeComponent{
		cType: TupleComponent,
		tupleChildren: []*typeComponent{
			{cType: 99},
		},
	})
	assert.Regexp(t, "FF22041", err)
}

func TestDecodeABIElementBadDynamicTypeFixedArray(t *testing.T) {

	block, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020")
	assert.NoError(t, err)

	_, _, err = decodeABIElement(context.Background(), "", block, 0, 0, &typeComponent{
		cType:       FixedArrayComponent,
		arrayLength: 1,
		arrayChild:  &typeComponent{cType: 99},
	})
	assert.Regexp(t, "FF22041", err)
}

func TestDecodeABIElementInsufficientDataFixedArrayDynamicType(t *testing.T) {

	p := &ParameterArray{
		{Type: "string[1]"},
	}
	tc, err := p.TypeComponentTree()
	assert.NoError(t, err)

	block, err := hex.DecodeString("00")
	assert.NoError(t, err)

	_, _, err = decodeABIElement(context.Background(), "", block, 0, 0, tc.(*typeComponent).tupleChildren[0])
	assert.Regexp(t, "FF22045", err)
}

func TestDecodeABIElementInsufficientDataTuple(t *testing.T) {

	p := &ParameterArray{
		{Type: "tuple", Components: ParameterArray{
			{Type: "string"},
		}},
	}
	tc, err := p.TypeComponentTree()
	assert.NoError(t, err)

	block, err := hex.DecodeString("00")
	assert.NoError(t, err)

	_, _, err = decodeABIElement(context.Background(), "", block, 0, 0, tc.(*typeComponent).tupleChildren[0])
	assert.Regexp(t, "FF22045", err)
}
