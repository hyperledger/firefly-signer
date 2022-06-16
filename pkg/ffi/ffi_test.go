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

package ffi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/stretchr/testify/assert"
)

func TestGetFFIType(t *testing.T) {
	assert.Equal(t, InputTypeString, GetFFIType("string"))
	assert.Equal(t, InputTypeString, GetFFIType("address"))
	assert.Equal(t, InputTypeString, GetFFIType("byte"))
	assert.Equal(t, InputTypeBoolean, GetFFIType("bool"))
	assert.Equal(t, InputTypeInteger, GetFFIType("uint256"))
	assert.Equal(t, InputTypeObject, GetFFIType("tuple"))
	assert.Equal(t, fftypes.FFEnumValue("", ""), GetFFIType("foobar"))
}

func TestFFIMethodToABI(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"details": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "newValue",
				Type:    "uint256",
				Indexed: false,
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)
	// assert.Equal(t, expectedABIElement, abi)

	actualABIJSON, err := json.Marshal(abi)
	assert.NoError(t, err)
	expectedABIJSON, err := json.Marshal(expectedABIElement)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIJSON), string(actualABIJSON))
}

func TestFFIMethodToABIObject(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "widget",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "object",
					"details": {
						"type": "tuple"
					},
					"properties": {
						"radius": {
							"type": "integer",
							"details": {
								"type": "uint256",
								"index": 0,
								"indexed": true
							}
						},
						"numbers": {
							"type": "array",
							"details": {
								"type": "uint256[]",
								"index": 1
							},
							"items": {
								"type": "integer",
								"details": {
									"type": "uint256"
								}
							}
						}
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "widget",
				Type:    "tuple",
				Indexed: false,
				Components: abi.ParameterArray{
					{
						Name:         "radius",
						Type:         "uint256",
						Indexed:      true,
						InternalType: "",
					},
					{
						Name:         "numbers",
						Type:         "uint256[]",
						Indexed:      false,
						InternalType: "",
					},
				},
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)
	assert.ObjectsAreEqual(expectedABIElement, abi)
}

func TestABIFFIConversionArrayOfObjects(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"components": [
						{
							"internalType": "string",
							"name": "name",
							"type": "string"
						},
						{
							"internalType": "uint256",
							"name": "weight",
							"type": "uint256"
						},
						{
							"internalType": "uint256",
							"name": "volume",
							"type": "uint256"
						},
						{
							"components": [
								{
									"internalType": "string",
									"name": "name",
									"type": "string"
								},
								{
									"internalType": "string",
									"name": "description",
									"type": "string"
								},
								{
									"internalType": "enum ComplexStorage.Alignment",
									"name": "alignment",
									"type": "uint8"
								}
							],
							"internalType": "struct ComplexStorage.BoxContent[]",
							"name": "contents",
							"type": "tuple[]"
						}
					],
					"internalType": "struct ComplexStorage.Box[]",
					"name": "newBox",
					"type": "tuple[]"
				}
			],
			"name": "set",
			"outputs": [],
			"stateMutability": "payable",
			"type": "function",
			"payable": true,
			"constant": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	abiFunction := abi.Functions()["set"]

	ffiMethod, err := convertABIFunctionToFFIMethod(context.Background(), abiFunction)
	assert.NoError(t, err)
	abiFunctionOut, err := ConvertFFIMethodToABI(context.Background(), ffiMethod)
	assert.NoError(t, err)

	expectedABIFunctionJSON, err := json.Marshal(abiFunction)
	assert.NoError(t, err)
	abiFunctionJSON, err := json.Marshal(abiFunctionOut)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedABIFunctionJSON), string(abiFunctionJSON))

}

func TestFFIMethodToABINestedArray(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "widget",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "array",
					"details": {
						"type": "string[][]",
						"internalType": "string[][]"
					},
					"items": {
						"type": "array",
						"items": {
							"type": "string"
						}
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:         "widget",
				Type:         "string[][]",
				InternalType: "string[][]",
				Indexed:      false,
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)
	expectedABIJSON, err := json.Marshal(expectedABIElement)
	assert.NoError(t, err)
	abiJSON, err := json.Marshal(abi)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedABIJSON), string(abiJSON))
}

func TestFFIMethodToABIInvalidJSON(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name:   "newValue",
				Schema: fftypes.JSONAnyPtr(`{#!`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "invalid character", err)
}

func TestFFIMethodToABIBadSchema(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"detailz": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "FF22052", err)
}

func TestFFIMethodToABIBadReturn(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name:   "set",
		Params: []*fftypes.FFIParam{},
		Returns: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"detailz": {
						"type": "uint256"
					}
				}`),
			},
		},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "FF22052", err)
}

func TestConvertABIToFFI(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "uint256",
					InternalType: "uint256",
				},
			},
			Outputs: abi.ParameterArray{},
		},
		{
			Name:   "get",
			Type:   "function",
			Inputs: abi.ParameterArray{},
			Outputs: abi.ParameterArray{
				{
					Name:         "value",
					Type:         "uint256",
					InternalType: "uint256",
				},
			},
		},
		{
			Name: "Updated",
			Type: "event",
			Inputs: abi.ParameterArray{{
				Name:         "value",
				Type:         "uint256",
				InternalType: "uint256",
			}},
			Outputs: abi.ParameterArray{},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"integer","details":{"type":"uint256","internalType":"uint256"}}`)

	expectedFFI := &fftypes.FFI{
		Name:        "SimpleStorage",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
			},
			{
				Name:   "get",
				Params: fftypes.FFIParams{},
				Returns: fftypes.FFIParams{
					{
						Name:   "value",
						Schema: schema,
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "Updated",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: schema,
						},
					},
				},
			},
		},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "SimpleStorage", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	assert.ObjectsAreEqual(expectedFFI, actualFFI)
}

func TestConvertABIToFFIWithObject(t *testing.T) {
	abi := &abi.ABI{
		&abi.Entry{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "tuple",
					InternalType: "struct WidgetFactory.Widget",
					Components: abi.ParameterArray{
						{
							Name:         "size",
							Type:         "uint256",
							InternalType: "uint256",
						},
						{
							Name:         "description",
							Type:         "string",
							InternalType: "string",
						},
					},
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	bigIntDesc := i18n.Expand(context.Background(), signermsgs.APIIntegerDescription)
	schema := fftypes.JSONAnyPtr(fmt.Sprintf(`{"type":"object","details":{"type":"tuple","internalType":"struct WidgetFactory.Widget"},"properties":{"description":{"type":"string","details":{"type":"string","internalType":"string","index":1}},"size":{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256","internalType":"uint256","index":0},"description":"%s"}}}`, bigIntDesc))

	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)

	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))

}

func TestConvertABIToFFIWithArray(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "string[]",
					InternalType: "string[]",
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"array","details":{"type":"string[]","internalType":"string[]"},"items":{"type":"string"}}`)

	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)

	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIWithNestedArray(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "uint256[][]",
					InternalType: "uint256[][]",
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"array","details":{"type":"uint256[][]","internalType":"uint256[][]"},"items":{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"integer"}],"description":"An integer. You are recommended to use a JSON string. A JSON number can be used for values up to the safe maximum."}}}`)
	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIWithNestedArrayOfObjects(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					InternalType: "struct WidgetFactory.Widget[][]",
					Name:         "gears",
					Type:         "tuple[][]",
					Components: abi.ParameterArray{
						{
							InternalType: "string",
							Name:         "description",
							Type:         "string",
						},
						{
							InternalType: "uint256",
							Name:         "size",
							Type:         "uint256",
						},
						{
							InternalType: "bool",
							Name:         "inUse",
							Type:         "bool",
						},
					},
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	bigIntDesc := i18n.Expand(context.Background(), signermsgs.APIIntegerDescription)
	schema := fftypes.JSONAnyPtr(fmt.Sprintf(`{"type":"array","details":{"type":"tuple[][]","internalType":"struct WidgetFactory.Widget[][]"},"items":{"type":"array","items":{"type":"object","properties":{"description":{"type":"string","details":{"type":"string","internalType":"string","index":0}},"inUse":{"type":"boolean","details":{"type":"bool","internalType":"bool","index":2}},"size":{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256","internalType":"uint256","index":1},"description":"%s"}}}}}`, bigIntDesc))
	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "gears",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIBadInputType(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"outputs": [],
			"stateMutability": "payable",
			"type": "function",
			"payable": true,
			"constant": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIToFFIBadOutputType(t *testing.T) {
	abiJSON := `[
		{
			"outputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"stateMutability": "viewable",
			"type": "function"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIToFFIBadEventType(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"type": "event"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIEventFFIEvent(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "string"
				}
			],
			"name": "set",
			"type": "event",
			"anonymous": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	ffi, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.NoError(t, err)

	actualABIEvent, err := ConvertFFIEventDefinitionToABI(context.Background(), &ffi.Events[0].FFIEventDefinition)
	assert.NoError(t, err)

	expectedABIEventJSON, err := json.Marshal(abi.Events()["set"])
	assert.NoError(t, err)
	actualABIEventJSON, err := json.Marshal(actualABIEvent)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIEventJSON), string(actualABIEventJSON))
}
