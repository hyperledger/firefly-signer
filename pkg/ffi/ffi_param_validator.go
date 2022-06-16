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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type ParamValidator struct{}

var intRegex, _ = regexp.Compile("^u?int([0-9]{1,3})$")
var bytesRegex, _ = regexp.Compile("^bytes([0-9]{1,2})?")

func (v *ParamValidator) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	valid := true
	if details, ok := m["details"]; ok {
		var jsonTypeString string
		n, _ := details.(map[string]interface{})
		blockchainType := n["type"].(string)
		jsonType, ok := m["type"]
		if ok {
			jsonTypeString = jsonType.(string)
		} else {
			_, ok := m["oneOf"]
			if ok {
				jsonTypeString = integerType
			}
		}
		switch jsonTypeString {
		case stringType:
			if blockchainType != stringType &&
				blockchainType != addressType &&
				!isEthereumNumberType(blockchainType) &&
				!isEthereumBytesType(blockchainType) {
				valid = false
			}
		case integerType:
			if !isEthereumNumberType(blockchainType) {
				valid = false
			}
		case booleanType:
			if blockchainType != boolType {
				valid = false
			}
		case arrayType:
			if !strings.HasSuffix(blockchainType, "[]") {
				valid = false
			}
		case objectType:
			if blockchainType != tupleType {
				valid = false
			}
		}

		if valid {
			return detailsSchema(n), nil
		}
		return nil, fmt.Errorf("cannot cast %v to %v", jsonType, blockchainType)
	}
	return nil, nil
}

func (v *ParamValidator) GetMetaSchema() *jsonschema.Schema {
	return jsonschema.MustCompileString("ffiParamDetails.json", `{
		"$ref": "#/$defs/ethereumParam",
		"$defs": {
			"ethereumParam": {
				"oneOf": [
					{
						"type": "object",
						"properties": {
							"type": {
								"type": "string",
								"not": {
									"const": "object"
								}
							},
							"details": {
								"$ref": "#/$defs/details"
							}
						},
						"required": [
							"details"
						]
					},
					{
						"type": "object",
						"properties": {
							"type": {
								"const": "object"
							},
							"details": {
								"$ref": "#/$defs/details"
							},
							"properties": {
								"type": "object",
								"patternProperties": {
									".*": {
										"$ref": "#/$defs/ethereumObjectChildParam"
									}
								}
							}
						},
						"required": [
							"details",
							"type"
						]
					}
				]
			},
			"ethereumObjectChildParam": {
				"oneOf": [
					{
						"type": "object",
						"properties": {
							"type": {
								"type": "string",
								"not": {
									"const": "object"
								}
							},
							"details": {
								"$ref": "#/$defs/objectFieldDetails"
							}
						},
						"required": [
							"details"
						]
					},
					{
						"type": "object",
						"properties": {
							"type": {
								"const": "object"
							},
							"details": {
								"$ref": "#/$defs/objectFieldDetails"
							},
							"properties": {
								"type": "object",
								"patternProperties": {
									".*": {
										"$ref": "#/$defs/ethereumObjectChildParam"
									}
								}
							}
						},
						"required": [
							"details"
						]
					}
				]
			},
			"details": {
				"type": "object",
				"properties": {
					"type": {
						"type": "string"
					},
					"internalType": {
						"type": "string"
					},
					"indexed": {
						"type": "boolean"
					}
				},
				"required": [
					"type"
				]
			},
			"objectFieldDetails": {
				"type": "object",
				"properties": {
					"type": {
						"type": "string"
					},
					"internalType": {
						"type": "string"
					},
					"indexed": {
						"type": "boolean"
					},
					"index": {
						"type": "integer"
					}
				},
				"required": [
					"type",
					"index"
				]
			}
		}
	}`)
}

func (v *ParamValidator) GetExtensionName() string {
	return "details"
}

type detailsSchema map[string]interface{}

func (s detailsSchema) Validate(ctx jsonschema.ValidationContext, v interface{}) error {
	// TODO: Additional validation of actual input possible in the future
	return nil
}

func isEthereumNumberType(input string) bool {
	matches := intRegex.FindStringSubmatch(input)
	if len(matches) == 2 {
		i, err := strconv.ParseInt(matches[1], 10, 0)
		if err == nil && i >= 8 && i <= 256 && i%8 == 0 {
			// valid
			return true
		}
	}
	return false
}

func isEthereumBytesType(input string) bool {
	matches := bytesRegex.FindStringSubmatch(input)
	if len(matches) == 2 {
		if matches[1] == "" {
			return true
		}
		i, err := strconv.ParseInt(matches[1], 10, 0)
		if err == nil && i >= 1 && i <= 32 {
			// valid
			return true
		}
	}
	return false
}
