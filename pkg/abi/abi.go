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

/*

The abi package allows encoding and decoding of ABI encoded bytes, for the inputs/outputs
to EVM functions, and the parsing of EVM logs/events.

A high level summary of the API is as follows:

                         [ ABI ]        - parse your ABI definition, using the Go model of the JSON format
                            ↓
                        (validate)      - all types in functions (methods), events and errors are validated
                            ↓
                [ ComponentType tree ]  - to build a "type tree" of all the arrays/tuples/elementary
                            ↓
    [ JSON ] →  [ ComponentValue tree ] - which you combine with data (JSON or Go types) to get a "value tree"
                            ↓
                         (encode)       - the value tree can then be serialized into ABI encoded bytes
                            ↓
                  [ ABI encoded bytes ] - so you can use these bytes to invoke EVM functions (signatures supported)
                            ↓
                         (decode)       - then you can decode ABI bytes from function outputs, or logs (event data)
                            ↓
    [ JSON ] ← [ ComponentValue tree ]  - the value tree can be serialized back to JSON

Example:

	transferABI := `[
		{
			"inputs": [
				{
					"internalType": "address",
					"name": "recipient",
					"type": "address"
				},
				{
					"internalType": "uint256",
					"name": "amount",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"internalType": "bool",
					"name": "",
					"type": "bool"
				}
			],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`

	// Parse the ABI definition
	var abi ABI
	_ = json.Unmarshal([]byte(transferABI), &abi)
	f := abi.Functions()["transfer"]

	// Parse some JSON input data conforming to the ABI
	encodedValueTree, _ := f.Inputs.ParseJSON([]byte(`{
		"recipient": "0x03706Ff580119B130E7D26C5e816913123C24d89",
		"amount": "1000000000000000000"
	}`))

	// We can serialize this directly to abi bytes
	abiData, _ := encodedValueTree.EncodeABIData()
	fmt.Println(hex.EncodeToString(abiData))
	// 00000000000000000000000003706ff580119b130e7d26c5e816913123c24d890000000000000000000000000000000000000000000000000de0b6b3a7640000

	// We can also serialize that to function call data, with the function selector prefix
	abiCallData, _ := f.EncodeCallData(encodedValueTree)

	// Decode those ABI bytes back again, verifying the function selector
	decodedValueTree, _ := f.DecodeABIInputs(abiCallData)

	// Serialize back to JSON
	jsonData, _ := decodedValueTree.JSON()

	// Output
	fmt.Println(string(jsonData))
	// {"amount":"1000000000000000000","recipient":"03706ff580119b130e7d26c5e816913123c24d89"}

The package deliberately gives you access to perform all of the transitions individually.

For example, if you want to traverse the type tree itself to generate metadata for the ABI, you can do that.

External data parsing tries to be flexible when coercing JSON data into a value tree:

- Bytes and Addresses can be any of:
  - Hex string without any prefix
  - Hex string with an "0x" prefix
  - A byte array
- Numbers can be any of:
  - A base10 formatted string without any prefix
  - A hex formatted string with an "0x" prefix
  - A number
  - Negative numbers are supported
  - Floating point numbers are supported (for ABI fixed/ufixed types)
- Boolean values can be any of:
  - A boolean
  - A string "true"/"false"
- Strings must be a string

When passing in an interface{} (instead of JSON directly) efforts are made to follow pointers,
and resolve types down to the basic types. For example detecting whether a struct conforms to
the fmt.Stringer interface.

For serialization back out from the value tree, to JSON, there is a pluggable formatting interface
with a number of built-in options as follows:

- Parameter serialization for function outputs / event log data (and nested tuples) can be:
  - Object based {"key1":"val1"}
  - Flat ordered array based ["val1"]
  - Self describing array based [{"name":"key1","type":"string","value":"val1"}]
- Number serialization can be:
  - Base 10 formatted string
  - Hex with "0x" prefix
  - Numeric up to the maximum safe Javscript values, then automatically switching to string
- Byte serialization can be:
  - Hex with "0x" prefix
  - Hex without any prefix
*/
package abi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"golang.org/x/crypto/sha3"
)

// ABI "Application Binary Interface" is a list of the methods and events
// on the external interface of an EVM based smart contract - written in
// Solidity / Vyper.
//
// It is structured as a JSON array of ABI entries, each of which can be
// a function, event or error definition.
type ABI []*Entry

// EntryType is an enum of the possible ABI entry types
type EntryType string

const (
	Function    EntryType = "function"    // A function/method of the smart contract
	Constructor EntryType = "constructor" // The constructor
	Receive     EntryType = "receive"     // The "receive Ethere" function
	Fallback    EntryType = "fallback"    // The default function to invoke
	Event       EntryType = "event"       // An event the smart contract can emit
	Error       EntryType = "error"       // An error definition
)

type StateMutability string

const (
	Pure       StateMutability = "pure"       // Specified not to read blockchain state
	View       StateMutability = "view"       // Specified not to modify the blockchain state (read-only)
	Payable    StateMutability = "payable"    // The function accepts ether
	NonPayable StateMutability = "nonpayable" // The function does not accept ether
)

type ParameterArray []*Parameter

// Entry is an individual entry in an ABI - a function, event or error.
//
// Defines the name / inputs / outputs which can be used to generate the signature
// of the function/event, and used to encode input data, or decode output data.
type Entry struct {
	Type            EntryType       `json:"type,omitempty"`            // Type of the entry - there are multiple function sub-types, events and errors
	Name            string          `json:"name,omitempty"`            // Name of the function/event/error
	Payable         bool            `json:"payable,omitempty"`         // Functions only: Superseded by stateMutability payable/nonpayable
	Constant        bool            `json:"constant,omitempty"`        // Functions only: Superseded by stateMutability pure/view
	Anonymous       bool            `json:"anonymous,omitempty"`       // Events only: The event is emitted without a signature (topic[0] is not generated)
	StateMutability StateMutability `json:"stateMutability,omitempty"` // How the function interacts with the blockchain state
	Inputs          ParameterArray  `json:"inputs"`                    // The list of input parameters to a function, or fields of an event / error
	Outputs         ParameterArray  `json:"outputs"`                   // Functions only: The list of return values from a function
}

// Parameter is an individual typed parameter input/output
type Parameter struct {
	Name         string         `json:"name"`                   // The name of the argument - does not affect the signature
	Type         string         `json:"type"`                   // The canonical type of the parameter
	InternalType string         `json:"internalType,omitempty"` // Additional internal type information that might be generated by the compiler
	Components   ParameterArray `json:"components,omitempty"`   // An ordered list (tuple) of nested elements for array/object types
	Indexed      bool           `json:"indexed,omitempty"`      // Events only: Whether the parameter is indexed into one of the topics of the log, or in the log's data segment

	parsed *typeComponent // cached components
}

func (e *Entry) IsFunction() bool {
	switch e.Type {
	case Function, Constructor, Receive, Fallback:
		return true
	default:
		return false
	}
}

// Validate processes all the components of all the entries in this ABI, to build a parsing tree
func (a ABI) Validate() (err error) {
	return a.ValidateCtx(context.Background())
}

func (a ABI) ValidateCtx(ctx context.Context) (err error) {
	for _, e := range a {
		if err := e.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a ABI) Functions() map[string]*Entry {
	m := make(map[string]*Entry)
	for _, e := range a {
		if e.Name != "" && e.IsFunction() {
			m[e.Name] = e
		}
	}
	return m
}

func (a ABI) Events() map[string]*Entry {
	m := make(map[string]*Entry)
	for _, e := range a {
		if e.Name != "" && e.Type == Event {
			m[e.Name] = e
		}
	}
	return m
}

// Validate processes all the components of all the parameters in this ABI entry
func (e *Entry) Validate() (err error) {
	return e.ValidateCtx(context.Background())
}

func (e *Entry) ValidateCtx(ctx context.Context) (err error) {
	for _, input := range e.Inputs {
		if err := input.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	for _, output := range e.Outputs {
		if err := output.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ParseJSON takes external JSON data, and parses againt the ABI to generate
// a component value tree.
//
// The component value tree can then be serialized to binary ABI data.
func (pa ParameterArray) ParseJSON(data []byte) (*ComponentValue, error) {
	return pa.ParseJSONCtx(context.Background(), data)
}

func (pa ParameterArray) ParseJSONCtx(ctx context.Context, data []byte) (*ComponentValue, error) {
	var jsonTree interface{}
	err := json.Unmarshal(data, &jsonTree)
	if err != nil {
		return nil, err
	}
	return pa.ParseExternalDataCtx(ctx, jsonTree)
}

// ParseExternalData takes (non-ABI encoded) data input, such as an unmarshalled JSON structure,
// and traverses it against the ABI component type tree, to form a component value tree.
//
// The component value tree can then be serialized to binary ABI data.
func (pa ParameterArray) ParseExternalData(input interface{}) (cv *ComponentValue, err error) {
	return pa.ParseExternalDataCtx(context.Background(), input)
}

// TypeComponentTree returns the type component tree for the array (tuple) of individually typed parameters
func (pa ParameterArray) TypeComponentTree() (component TypeComponent, err error) {
	return pa.TypeComponentTreeCtx(context.Background())
}

func (pa ParameterArray) TypeComponentTreeCtx(ctx context.Context) (tc TypeComponent, err error) {
	component := &typeComponent{
		cType:         TupleComponent,
		tupleChildren: make([]*typeComponent, len(pa)),
	}
	for i, p := range pa {
		if component.tupleChildren[i], err = p.typeComponentTreeCtx(ctx); err != nil {
			return nil, err
		}
	}
	return component, nil
}

func (pa ParameterArray) ParseExternalDataCtx(ctx context.Context, input interface{}) (cv *ComponentValue, err error) {
	component, err := pa.TypeComponentTreeCtx(ctx)
	if err != nil {
		return nil, err
	}
	return walkInput(ctx, "", input, component.(*typeComponent))
}

// DecodeABIData takes ABI encoded bytes that conform to the parameter array, and decodes them
// into a value tree. We take the offset (rather than requiring you to generate a slice at the
// given offset) so that errors in parsing can be reported at an absolute offset.
func (pa ParameterArray) DecodeABIData(b []byte, offset int) (cv *ComponentValue, err error) {
	return pa.DecodeABIDataCtx(context.Background(), b, offset)
}

func (pa ParameterArray) DecodeABIDataCtx(ctx context.Context, b []byte, offset int) (cv *ComponentValue, err error) {
	component, err := pa.TypeComponentTreeCtx(ctx)
	if err != nil {
		return nil, err
	}
	_, cv, err = decodeABIElement(ctx, "", b, offset, offset, component.(*typeComponent))
	return cv, err
}

// String returns the signature string. If a Validate needs to be initiated, and that
// parse fails, then the error is logged, but is not returned
func (e *Entry) String() string {
	s, err := e.Signature()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
	}
	return s
}

func (e *Entry) Signature() (string, error) {
	return e.SignatureCtx(context.Background())
}

func (e *Entry) GenerateID() ([]byte, error) {
	return e.GenerateIDCtx(context.Background())
}

func (e *Entry) GenerateIDCtx(ctx context.Context) ([]byte, error) {
	hash := sha3.NewLegacyKeccak256()
	sig, err := e.SignatureCtx(ctx)
	if err != nil {
		return nil, err
	}
	hash.Write([]byte(sig))
	k := hash.Sum(nil)
	return k[0:4], nil
}

// ID is a convenience function to get the ID as a hex string (no 0x prefix), which will
// return the empty string on failure
func (e *Entry) ID() string {
	id, err := e.GenerateID()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
		return ""
	}
	return hex.EncodeToString(id)
}

// EncodeCallData serializes the inputs of the entry, prefixed with the function selector
func (e *Entry) EncodeCallData(cv *ComponentValue) ([]byte, error) {
	return e.EncodeCallDataCtx(context.Background(), cv)
}

func (e *Entry) EncodeCallDataCtx(ctx context.Context, cv *ComponentValue) ([]byte, error) {

	id, err := e.GenerateIDCtx(ctx)
	if err != nil {
		return nil, err
	}

	cvData, err := cv.EncodeABIDataCtx(ctx)
	if err != nil {
		return nil, err
	}

	data := make([]byte, len(id)+len(cvData))
	copy(data, id)
	copy(data[len(id):], cvData)
	return data, nil

}

func (e *Entry) DecodeABIInputs(b []byte) (*ComponentValue, error) {
	return e.DecodeABIInputsCtx(context.Background(), b)
}

func (e *Entry) DecodeABIInputsCtx(ctx context.Context, b []byte) (*ComponentValue, error) {

	id, err := e.GenerateIDCtx(ctx)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughtBytesABISignature)
	}
	if !bytes.Equal(id, b[0:4]) {
		return nil, i18n.NewError(ctx, signermsgs.MsgIncorrectABISignatureID, e.String(), hex.EncodeToString(id), hex.EncodeToString(b[0:4]))
	}

	return e.Inputs.DecodeABIDataCtx(ctx, b, 4)

}

func (e *Entry) SignatureCtx(ctx context.Context) (string, error) {
	buff := new(strings.Builder)
	buff.WriteString(e.Name)
	buff.WriteRune('(')
	for i, p := range e.Inputs {
		if i > 0 {
			buff.WriteRune(',')
		}
		s, err := p.SignatureStringCtx(ctx)
		if err != nil {
			return "", err
		}
		buff.WriteString(s)
	}
	buff.WriteRune(')')
	return buff.String(), nil
}

// Validate processes all the components of the type of this ABI parameter.
// - The elementary type
// - The fixed/variable length array dimensions
// - The tuple component types (recursively)
func (p *Parameter) Validate() (err error) {
	return p.ValidateCtx(context.Background())
}

func (p *Parameter) ValidateCtx(ctx context.Context) (err error) {
	p.parsed, err = p.parseABIParameterComponents(ctx)
	return err
}

// SignatureString generates and returns the signature string of the ABI
// parameter. If Validate has not yet been called, it will be called on your behalf.
//
// Note if you have modified the structure since Validate was last called, you should
// call Validate again.
func (p *Parameter) SignatureString() (s string, err error) {
	return p.SignatureStringCtx(context.Background())
}

func (p *Parameter) SignatureStringCtx(ctx context.Context) (string, error) {
	// Ensure the type component tree has been parsed
	tc, err := p.TypeComponentTreeCtx(ctx)
	if err != nil {
		return "", err
	}
	return tc.String(), nil
}

// String returns the signature string. If a Validate needs to be initiated, and that
// parse fails, then the error is logged, but is not returned
func (p *Parameter) String() string {
	s, err := p.SignatureString()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
	}
	return s
}

// ComponentTypeTree returns the root of the component tree for the parameter.
// If Validate has not yet been called, it will be called on your behalf.
//
// Note if you have modified the structure since Validate was last called, you should
// call Validate again.
func (p *Parameter) TypeComponentTree() (TypeComponent, error) {
	return p.TypeComponentTreeCtx(context.Background())
}

func (p *Parameter) TypeComponentTreeCtx(ctx context.Context) (TypeComponent, error) {
	tc, err := p.typeComponentTreeCtx(ctx)
	return TypeComponent(tc), err
}

func (p *Parameter) typeComponentTreeCtx(ctx context.Context) (*typeComponent, error) {
	if p.parsed == nil {
		if err := p.ValidateCtx(ctx); err != nil {
			return nil, err
		}
	}
	return p.parsed, nil
}
