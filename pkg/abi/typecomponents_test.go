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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElementalTypeInfoRules(t *testing.T) {

	assert.Equal(t, "int<M> (8 <= M <= 256) (M mod 8 == 0) (int == int256)", ElementaryTypeInt.String())
	assert.Equal(t, "uint<M> (8 <= M <= 256) (M mod 8 == 0) (uint == uint256)", ElementaryTypeUint.String())
	assert.Equal(t, "address", ElementaryTypeAddress.String())
	assert.Equal(t, "bool", ElementaryTypeBool.String())
	assert.Equal(t, "fixed<M>x<N> (8 <= M <= 256) (1 <= N <= 80) (M mod 8 == 0) (fixed == fixed128x18)", ElementaryTypeFixed.String())
	assert.Equal(t, "ufixed<M>x<N> (8 <= M <= 256) (1 <= N <= 80) (M mod 8 == 0) (ufixed == ufixed128x18)", ElementaryTypeUfixed.String())
	assert.Equal(t, "bytes / bytes<M> (1 <= M <= 32)", ElementaryTypeBytes.String())
	assert.Equal(t, "function", ElementaryTypeFunction.String())
	assert.Equal(t, "string", ElementaryTypeString.String())
	assert.Equal(t, JSONEncodingTypeString, ElementaryTypeString.JSONEncodingType())
	assert.NotNil(t, ElementaryTypeString.DataReader())

}

func TestABISignatures1(t *testing.T) {

	abiString := `[{
			"type":"error",
			"inputs": [{"name":"available","type":"uint256"},{"name":"required","type":"uint256"}],
			"name":"InsufficientBalance"
		}, {
			"type":"event",
			"inputs": [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"bytes32","indexed":false}],
			"name":"Event"
		}, {
			"type":"event",
			"inputs": [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"bytes32","indexed":false}],
			"name":"Event2"
		}, {
			"type":"function",
			"inputs": [{"name":"a","type":"uint256"}],
			"name":"foo",
			"outputs": []
	    }
	]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)
	assert.NotNil(t, abi.Events()["Event2"])
	assert.Nil(t, abi.Functions()["Event2"])
	assert.NotNil(t, abi.Errors()["InsufficientBalance"])
	assert.Nil(t, abi.Functions()["InsufficientBalance"])

	sig0, err := abi[0].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "InsufficientBalance(uint256,uint256)", sig0)
	assert.False(t, abi[0].IsFunction())

	sig1, err := abi[1].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Event(uint256,bytes32)", sig1)
	assert.False(t, abi[1].IsFunction())

	sig2, err := abi[2].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Event2(uint256,bytes32)", sig2)
	assert.False(t, abi[1].IsFunction())

	sig3, err := abi[3].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "foo(uint256)", sig3)
	assert.True(t, abi[3].IsFunction())

}

func TestABISignatures2(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
			  "name": "s",
			  "type": "tuple",
			  "components": [
				{
				  "name": "a",
				  "type": "uint256"
				},
				{
				  "name": "b",
				  "type": "uint256[]"
				},
				{
				  "name": "c",
				  "type": "tuple[]",
				  "components": [
					{
					  "name": "x",
					  "type": "uint256"
					},
					{
					  "name": "y",
					  "type": "uint256"
					}
				  ]
				}
			  ]
			},
			{
			  "name": "t",
			  "type": "tuple",
			  "components": [
				{
				  "name": "x",
				  "type": "uint256"
				},
				{
				  "name": "y",
				  "type": "uint256"
				}
			  ]
			},
			{
			  "name": "a",
			  "type": "uint256"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	sig0, err := abi[0].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "f((uint256,uint256[],(uint256,uint256)[]),(uint256,uint256),uint256)", sig0)

}

func TestABISignatures3(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256[8][][16]"
			},
			{
				"name": "b",
				"type": "fixed"
			},
			{
			  "name": "c",
			  "type": "ufixed64x50[][][]"
			},
			{
			  "name": "d",
			  "type": "bytes"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	sig0, err := abi[0].Signature()
	assert.NoError(t, err)
	assert.Equal(t, "f(uint256[8][][16],fixed128x18,ufixed64x50[][][],bytes)", sig0)

}

func TestABIUnknownType(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "lobster"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22025.*lobster", err)

}

func TestABIInvalidArrayInNested(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "tuple",
				"components": [
				  {
					"name": "x",
					"type": "uint{}"
				  }
				]
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028.*uint{}", err)

}

func TestABIUnexpectedSuffix(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "address256"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22026", err)

}

func TestABIMissingMSuffix(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "unittest"
			}
		  ],
		  "outputs": []
		}
	  ]`
	registerElementaryType(elementaryTypeInfo{
		name:       "unittest",
		suffixType: suffixTypeMRequired,
	})
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22027", err)

}

func TestABIMissingMxNSuffix(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "unittest"
			}
		  ],
		  "outputs": []
		}
	  ]`
	registerElementaryType(elementaryTypeInfo{
		name:       "unittest",
		suffixType: suffixTypeMxNRequired,
	})
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22027", err)

}

func TestABIBadMSuffixTooBig(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint257"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadMSuffixTooSmall(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint0"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadMSuffixNonDecimal(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint0a"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadMSuffixNotMod8(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint9"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadNSuffixTooBig(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "ufixed128x81"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadNSuffixTooSmall(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "fixed128x0"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadMInMxNSuffixNonDecimal(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "fixed0fx1"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIMissingNInMxNSuffixNonDecimal(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "fixed128x"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadNSuffixNonDecimal(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "fixed128x8f"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadNSuffixOptionalTooBig(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "bytes33"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadNSuffixOptionalTooSmall(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "bytes0"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22028", err)

}

func TestABIBadArrayMDimension(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256[0f]"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22029", err)

}

func TestABIMissingArrayClose(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256["
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22029", err)

}

func TestABIMismatchedArrayClose(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256[]]"
			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	_, err = abi[0].Signature()
	assert.Regexp(t, "FF22029", err)

}

func TestTypeComponentStringInvalid(t *testing.T) {

	tc := &typeComponent{
		cType: ComponentType(-1),
	}
	assert.Empty(t, tc.String())

	isRef, solDef, childStructs := tc.SolidityTypeDef()
	assert.False(t, isRef)
	assert.Empty(t, solDef)
	assert.Empty(t, childStructs)

}

func TestTypeComponentParseExternalOk(t *testing.T) {

	tc := &typeComponent{
		cType:          ElementaryComponent,
		elementaryType: ElementaryTypeString.(*elementaryTypeInfo),
	}
	cv, err := tc.ParseExternal("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", cv.Value)

}

func TestTypeInternalTypeIndexed(t *testing.T) {

	abiString := `[
		{
		  "name": "f",
		  "type": "function",
		  "inputs": [
			{
				"name": "a",
				"type": "uint256[]",
				"internalType": "uint256[]",
				"indexed": true

			}
		  ],
		  "outputs": []
		}
	  ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)
	err = abi.Validate()
	assert.NoError(t, err)

	tc, err := abi.Functions()["f"].Inputs[0].TypeComponentTree()
	assert.NoError(t, err)
	assert.Equal(t, "uint256[]", tc.Parameter().Type)
	assert.Equal(t, "uint256[]", tc.Parameter().InternalType)
	assert.Equal(t, true, tc.Parameter().Indexed)
}

func TestDecodeABIDataOnNonTuple(t *testing.T) {

	_, err := (&typeComponent{}).DecodeABIData([]byte{}, 0)
	assert.Regexp(t, "FF22061", err)
}
