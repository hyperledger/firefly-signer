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

package ffi2abi

import (
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type ParamValidator struct{}

var compiledMetaSchema = jsonschema.MustCompileString("ffiParamDetails.json", `{
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
						"type",
						"details"
					]
				},
				{
					"type": "object",
					"properties": {
						"oneOf": {
							"type": "array"
						},
						"details": {
							"$ref": "#/$defs/details"
						}
					},
					"required": [
						"oneOf",
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
						"type",
						"details"
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
						"type",
						"details"
					]
				},
				{
					"type": "object",
					"properties": {
						"oneOf": {
							"type": "array"
						},
						"details": {
							"$ref": "#/$defs/objectFieldDetails"
						}
					},
					"required": [
						"oneOf",
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
						"type",
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

func (v *ParamValidator) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	return nil, nil
}

func (v *ParamValidator) GetMetaSchema() *jsonschema.Schema {
	return compiledMetaSchema
}

func (v *ParamValidator) GetExtensionName() string {
	return "details"
}
