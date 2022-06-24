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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/abi"
)

const (
	jsonBooleanType = "boolean"
	jsonIntegerType = "integer"
	jsonNumberType  = "number"
	jsonStringType  = "string"
	jsonArrayType   = "array"
	jsonObjectType  = "object"
)

// InputType is the type of a JSON field in a request to FireFly's API
type InputType = fftypes.FFEnum

var (
	// InputTypeInteger is a json integer or string to be treated as an integer
	InputTypeInteger = fftypes.FFEnumValue("ffiinputtype", "integer")
	// InputTypeNumber is a json number or string to be treated as an integer
	InputTypeNumber = fftypes.FFEnumValue("ffiinputtype", "number")
	// InputTypeString is a JSON string
	InputTypeString = fftypes.FFEnumValue("ffiinputtype", "string")
	// FFIInputTypeArray is a JSON boolean
	InputTypeBoolean = fftypes.FFEnumValue("ffiinputtype", "boolean")
	// InputTypeArray is a JSON array
	InputTypeArray = fftypes.FFEnumValue("ffiinputtype", "array")
	// InputTypeObject is a JSON object
	InputTypeObject = fftypes.FFEnumValue("ffiinputtype", "object")
)

type paramDetails struct {
	Type         string `json:"type"`
	InternalType string `json:"internalType,omitempty"`
	Indexed      bool   `json:"indexed,omitempty"`
	Index        *int   `json:"index,omitempty"`
}

type Schema struct {
	OneOf       []SchemaType       `json:"oneOf,omitempty"`
	Type        *fftypes.JSONAny   `json:"type,omitempty"`
	Details     *paramDetails      `json:"details,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Description string             `json:"description,omitempty"`
}

type SchemaType struct {
	Type string `json:"type"`
}

func (s *Schema) ToJSON() string {
	b, _ := json.Marshal(s)
	return string(b)
}

func ConvertFFIMethodToABI(ctx context.Context, method *fftypes.FFIMethod) (*abi.Entry, error) {
	abiInputs, err := convertFFIParamsToABIParameters(ctx, method.Params)
	if err != nil {
		return nil, err
	}

	abiOutputs, err := convertFFIParamsToABIParameters(ctx, method.Returns)
	if err != nil {
		return nil, err
	}
	abiEntry := &abi.Entry{
		Name:    method.Name,
		Type:    "function",
		Inputs:  abiInputs,
		Outputs: abiOutputs,
	}

	if method.Details != nil {
		if stateMutability, ok := method.Details.GetStringOk("stateMutability"); ok {
			abiEntry.StateMutability = abi.StateMutability(stateMutability)
		}
		abiEntry.Payable = method.Details.GetBool("payable")
		abiEntry.Constant = method.Details.GetBool("constant")
	}
	return abiEntry, nil
}

func ConvertFFIEventDefinitionToABI(ctx context.Context, event *fftypes.FFIEventDefinition) (*abi.Entry, error) {
	abiInputs, err := convertFFIParamsToABIParameters(ctx, event.Params)
	if err != nil {
		return nil, err
	}
	abiEntry := &abi.Entry{
		Name:   event.Name,
		Type:   "event",
		Inputs: abiInputs,
	}
	if event.Details != nil {
		abiEntry.Anonymous = event.Details.GetBool("anonymous")
	}
	return abiEntry, nil
}

func ConvertABIToFFI(ctx context.Context, ns, name, version, description string, abi *abi.ABI) (*fftypes.FFI, error) {
	ffi := &fftypes.FFI{
		Namespace:   ns,
		Name:        name,
		Version:     version,
		Description: description,
		Methods:     make([]*fftypes.FFIMethod, len(abi.Functions())),
		Events:      make([]*fftypes.FFIEvent, len(abi.Events())),
	}
	i := 0
	for _, f := range abi.Functions() {
		method, err := convertABIFunctionToFFIMethod(ctx, f)
		if err != nil {
			return nil, err
		}
		ffi.Methods[i] = method
		i++
	}
	i = 0
	for _, f := range abi.Events() {
		event, err := convertABIEventToFFIEvent(ctx, f)
		if err != nil {
			return nil, err
		}
		ffi.Events[i] = event
		i++
	}
	return ffi, nil
}

func convertABIFunctionToFFIMethod(ctx context.Context, abiFunction *abi.Entry) (*fftypes.FFIMethod, error) {
	params := make([]*fftypes.FFIParam, len(abiFunction.Inputs))
	returns := make([]*fftypes.FFIParam, len(abiFunction.Outputs))
	details := map[string]interface{}{}
	for i, input := range abiFunction.Inputs {
		typeComponent, err := input.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := getSchemaForABIInput(ctx, typeComponent)
		param := &fftypes.FFIParam{
			Name:   input.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		params[i] = param
	}
	for i, output := range abiFunction.Outputs {
		typeComponent, err := output.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := getSchemaForABIInput(ctx, typeComponent)
		param := &fftypes.FFIParam{
			Name:   output.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		returns[i] = param
	}
	if abiFunction.StateMutability != "" {
		details["stateMutability"] = string(abiFunction.StateMutability)
	}
	if abiFunction.Payable {
		details["payable"] = true
	}
	if abiFunction.Constant {
		details["constant"] = true
	}
	return &fftypes.FFIMethod{
		Name:    abiFunction.Name,
		Params:  params,
		Returns: returns,
		Details: details,
	}, nil
}

func convertABIEventToFFIEvent(ctx context.Context, abiEvent *abi.Entry) (*fftypes.FFIEvent, error) {
	params := make([]*fftypes.FFIParam, len(abiEvent.Inputs))
	details := map[string]interface{}{}
	for i, output := range abiEvent.Inputs {
		typeComponent, err := output.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := getSchemaForABIInput(ctx, typeComponent)
		param := &fftypes.FFIParam{
			Name:   output.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		params[i] = param
	}
	if abiEvent.Anonymous {
		details["anonymous"] = true
	}
	return &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name:    abiEvent.Name,
			Params:  params,
			Details: details,
		},
	}, nil
}

func getSchemaForABIInput(ctx context.Context, typeComponent abi.TypeComponent) *Schema {
	schema := &Schema{
		Details: &paramDetails{
			Type:         typeComponent.Parameter().Type,
			InternalType: typeComponent.Parameter().InternalType,
			Indexed:      typeComponent.Parameter().Indexed,
		},
	}
	switch typeComponent.ComponentType() {
	case abi.ElementaryComponent:
		switch typeComponent.ElementaryType().JSONEncodingType() {
		case abi.JSONEncodingTypeInteger:
			schema.OneOf = []SchemaType{
				{Type: jsonStringType},
				{Type: jsonIntegerType},
			}
			schema.Description = i18n.Expand(ctx, signermsgs.APIIntegerDescription)
		case abi.JSONEncodingTypeFloat:
			schema.OneOf = []SchemaType{
				{Type: jsonStringType},
				{Type: jsonNumberType},
			}
			schema.Description = i18n.Expand(ctx, signermsgs.APIFloatDescription)
		case abi.JSONEncodingTypeBool:
			schema.OneOf = []SchemaType{
				{Type: jsonStringType},
				{Type: jsonBooleanType},
			}
			schema.Description = i18n.Expand(ctx, signermsgs.APIBoolDescription)
		case abi.JSONEncodingTypeBytes:
			schema.Type = fftypes.JSONAnyPtr(fmt.Sprintf(`"%s"`, jsonStringType))
			schema.Description = i18n.Expand(ctx, signermsgs.APIHexDescription)
		default:
			schema.Type = fftypes.JSONAnyPtr(fmt.Sprintf(`"%s"`, jsonStringType))
		}
	case abi.FixedArrayComponent, abi.DynamicArrayComponent:
		schema.Type = fftypes.JSONAnyPtr(fmt.Sprintf(`"%s"`, jsonArrayType))
		childSchema := getSchemaForABIInput(ctx, typeComponent.ArrayChild())
		schema.Items = childSchema
		schema.Details = childSchema.Details
		childSchema.Details = nil
	case abi.TupleComponent:
		schema.Type = fftypes.JSONAnyPtr(fmt.Sprintf(`"%s"`, jsonObjectType))
		schema.Properties = make(map[string]*Schema, len(typeComponent.TupleChildren()))
		for i, tupleChild := range typeComponent.TupleChildren() {
			childSchema := getSchemaForABIInput(ctx, tupleChild)
			childSchema.Details.Index = new(int)
			*childSchema.Details.Index = i
			schema.Properties[tupleChild.KeyName()] = childSchema
		}
	}
	return schema
}

func convertFFIParamsToABIParameters(ctx context.Context, params fftypes.FFIParams) (abi.ParameterArray, error) {
	abiParamList := make(abi.ParameterArray, len(params))
	for i, param := range params {

		// We need to validate the schema against the Ethereum FFI metaschema here
		// For example, if someone is creating an event listener, we need to make
		// sure we have all the correct ABI fields. This does not validate function
		// arguments themselves. The blockchain connector or node will do that.

		c := fftypes.NewFFISchemaCompiler()
		v := &ParamValidator{}
		c.RegisterExtension(v.GetExtensionName(), v.GetMetaSchema(), v)
		err := c.AddResource(param.Name, strings.NewReader(param.Schema.String()))
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidFFIDetailsSchema, param.Name)
		}
		_, err = c.Compile(param.Name)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidFFIDetailsSchema, param.Name)
		}

		var s *Schema
		// Errors here are unchecked because they cannot be hit if the above JSON Schema validation passed
		_ = json.Unmarshal(param.Schema.Bytes(), &s)
		abiParameter, _ := processField(ctx, param.Name, s)

		tc, err := abiParameter.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidFFIDetailsSchema, param.Name)
		}
		if !inputTypeValidForTypeComponent(ctx, s.Type, tc) {
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidFFIDetailsSchema, param.Name)
		}

		abiParamList[i] = abiParameter
	}
	return abiParamList, nil
}

func inputTypeValidForTypeComponent(ctx context.Context, inputType *fftypes.JSONAny, tc abi.TypeComponent) bool {
	var inputTypeString string
	if err := inputType.Unmarshal(ctx, &inputTypeString); err != nil {
		if o, ok := inputType.JSONObjectOk(); ok {
			if a, ok := o.GetObjectArrayOk("oneOf"); ok {
				for _, t := range a {
					// Look for the entry in the oneOf that isn't "string"
					jsonType := t.GetString("type")
					if jsonType != "" && jsonType != inputTypeString {
						inputTypeString = jsonType
					}
				}
			}
		}
	}
	switch inputTypeString {
	case jsonBooleanType:
		// Booleans are only valid for boolean types
		return tc.ComponentType() == abi.ElementaryComponent && tc.ElementaryType().JSONEncodingType() == abi.JSONEncodingTypeBool
	case jsonIntegerType:
		// Integers are only valid for integer types
		return tc.ComponentType() == abi.ElementaryComponent && tc.ElementaryType().JSONEncodingType() == abi.JSONEncodingTypeInteger
	case jsonNumberType:
		// Integers are only valid for float types
		return tc.ComponentType() == abi.ElementaryComponent && tc.ElementaryType().JSONEncodingType() == abi.JSONEncodingTypeFloat
	case jsonStringType:
		// Strings are valid for all elementary components
		return tc.ComponentType() == abi.ElementaryComponent
	case jsonArrayType:
		return tc.ComponentType() == abi.DynamicArrayComponent || tc.ComponentType() == abi.FixedArrayComponent
	case jsonObjectType:
		return tc.ComponentType() == abi.TupleComponent
	}
	return false
}

func buildABIParameterArrayForObject(ctx context.Context, properties map[string]*Schema) (abi.ParameterArray, error) {
	parameters := make(abi.ParameterArray, len(properties))
	for propertyName, propertySchema := range properties {
		parameter, err := processField(ctx, propertyName, propertySchema)
		if err != nil {
			return nil, err
		}
		parameters[*propertySchema.Details.Index] = parameter
	}
	return parameters, nil
}

func processField(ctx context.Context, name string, schema *Schema) (*abi.Parameter, error) {
	if schema.Details == nil {
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidFFIDetailsSchema, name)
	}
	parameter := &abi.Parameter{
		Name:         name,
		Type:         schema.Details.Type,
		InternalType: schema.Details.InternalType,
		Indexed:      schema.Details.Indexed,
	}
	var schemaTypeString string
	if err := json.Unmarshal(schema.Type.Bytes(), &schemaTypeString); err == nil {
		switch schemaTypeString {
		case jsonObjectType:
			parameter.Components, err = buildABIParameterArrayForObject(ctx, schema.Properties)
		case jsonArrayType:
			parameter.Components, err = buildABIParameterArrayForObject(ctx, schema.Items.Properties)
		}
		if err != nil {
			return nil, i18n.WrapError(ctx, err, signermsgs.MsgInvalidFFIDetailsSchema, name)
		}
	}
	return parameter, nil
}

func ABIArgumentToTypeString(typeName string, components abi.ParameterArray) string {
	if strings.HasPrefix(typeName, "tuple") {
		suffix := typeName[5:]
		children := make([]string, len(components))
		for i, component := range components {
			children[i] = ABIArgumentToTypeString(component.Type, nil)
		}
		return "(" + strings.Join(children, ",") + ")" + suffix
	}
	return typeName
}

func ABIMethodToSignature(abi *abi.Entry) string {
	result := abi.Name + "("
	if len(abi.Inputs) > 0 {
		types := make([]string, len(abi.Inputs))
		for i, param := range abi.Inputs {
			types[i] = ABIArgumentToTypeString(param.Type, param.Components)
		}
		result += strings.Join(types, ",")
	}
	result += ")"
	return result
}
