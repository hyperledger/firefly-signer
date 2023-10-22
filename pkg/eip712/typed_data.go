// Copyright © 2023 Kaleido, Inc.
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

package eip712

import (
	"sort"
	"strings"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
)

type TypeMember struct {
	Name string
	Type string
}

type Type []*TypeMember

type SigningRequest struct {
	Types       TypeSet                `json:"types"`
	PrimaryType string                 `json:"primaryType"`
	Domain      Domain                 `json:"domain"`
	Message     map[string]interface{} `json:"message"`
}

type TypeSet map[string]Type

type Domain struct {
	Name              string                `json:"name"`
	Version           string                `json:"version"`
	ChainID           int64                 `json:"chainId"`
	VerifyingContract ethtypes.Address0xHex `json:"verifyingContract"`
}

// A map from type names to types is encoded per encodeType:
//
// > If the struct type references other struct types (and these in turn reference even more struct types),
// > then the set of referenced struct types is collected, sorted by name and appended to the encoding.
func (ts TypeSet) Encode(primaryType string) string {
	// Write the primary type first always
	buff := new(strings.Builder)
	buff.WriteString(ts[primaryType].Encode(primaryType))

	// Then the reference types sorted by name
	referenceTypes := make([]string, 0, len(ts))
	for typeName := range ts {
		if typeName != primaryType {
			referenceTypes = append(referenceTypes, typeName)
		}
	}
	sort.Strings(referenceTypes)
	for _, typeName := range referenceTypes {
		buff.WriteString(ts[typeName].Encode(typeName))
	}
	return buff.String()
}

// An individual member is encoded as:
//
// > type ‖ " " ‖ name
func (tm *TypeMember) Encode() string {
	return tm.Type + " " + tm.Name
}

// A type is encoded as:
//
// > name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"
func (t Type) Encode(name string) string {
	buff := new(strings.Builder)
	buff.WriteString(name)
	buff.WriteRune('(')
	for i, tm := range t {
		if i > 0 {
			buff.WriteRune(',')
		}
		buff.WriteString(tm.Encode())
	}
	buff.WriteRune(')')
	return buff.String()
}

// var EIP712DomainType = Type{
// 	Name: "EIP712Domain",
// 	Members: []*TypeMember{
// 		{
// 			Name: "name",
// 			Type: "string",
// 		},
// 		{
// 			Name: "version",
// 			Type: "string",
// 		},
// 		{
// 			Name: "chainId",
// 			Type: "uint256",
// 		},
// 		{
// 			Name: "verifyingContract",
// 			Type: "address",
// 		},
// 		{
// 			Name: "salt",
// 			Type: "bytes32",
// 		},
// 	},
// }

// func (t *TypeMember) Encode() string {
// 	return t.Type + " " + t.Name
// }

// func (t *Type) Encode() string {
// 	params := make([]string, len(t.Members))
// 	for i, member := range t.Members {
// 		params[i] = member.Encode()
// 	}
// 	return t.Name + "(" + strings.Join(params, ",") + ")"
// }

// func (t *Type) Hash() []byte {
// 	hash := sha3.NewLegacyKeccak256()
// 	hash.Write([]byte(t.Encode()))
// 	return hash.Sum(nil)
// }

// func encodeData(members []*TypeMember, data map[string]interface{}) ([]byte, error) {
// 	var result []byte
// 	for _, member := range members {
// 		val := data[member.Name]
// 		encodedVal, err := encodeValue(member, val)
// 		if err != nil {
// 			return nil, err
// 		}
// 		result = append(result, encodedVal...)
// 	}
// 	return result, nil
// }

// func encodeValue(m *TypeMember, val interface{}) ([]byte, error) {
// 	switch m.Type {
// 	case "bool":
// 		return encodeBool(val.(bool)), nil
// 	case "address":
// 		return encodeAddress(val.(string)), nil
// 	case "uint256":
// 		return encodeUintString(val.(string)), nil
// 	case "string":
// 		return encodeString(val.(string)), nil
// 	default:
// 		return nil, fmt.Errorf("unsupported type: %s", m.Type)
// 	}
// }

// func encodeBool(val bool) []byte {
// 	var i *big.Int
// 	if val {
// 		i = big.NewInt(1)
// 	} else {
// 		i = big.NewInt(0)
// 	}
// 	return encodeUint(i)
// }

// func encodeAddress(val string) []byte {
// 	i := big.NewInt(0)
// 	i.SetString(val, 16)
// 	return encodeUint(i)
// }

// func encodeUint(val *big.Int) []byte {
// 	data := make([]byte, 32)
// 	_ = val.FillBytes(data)
// 	return data
// }

// func encodeUintString(val string) []byte {
// 	i := big.NewInt(0)
// 	i.SetString(val, 10)
// 	return encodeUint(i)
// }

// func encodeString(val string) []byte {
// 	return encodeDynamicBytes([]byte(val))
// }

// func encodeDynamicBytes(val []byte) []byte {
// 	hash := sha3.NewLegacyKeccak256()
// 	hash.Write(val)
// 	return hash.Sum(nil)
// }

// func hashStruct(t *Type, data map[string]interface{}) ([]byte, error) {
// 	encodedData, err := encodeData(t.Members, data)
// 	if err != nil {
// 		return nil, err
// 	}
// 	hash := sha3.NewLegacyKeccak256()
// 	hash.Write(t.Hash())
// 	hash.Write(encodedData)
// 	return hash.Sum(nil), nil
// }

// func EncodeTypedData(domain map[string]interface{}, t *Type, message map[string]interface{}) ([]byte, error) {
// 	domainSeparator, err := hashStruct(&EIP712DomainType, domain)
// 	if err != nil {
// 		return nil, err
// 	}
// 	messageHash, err := hashStruct(t, message)
// 	if err != nil {
// 		return nil, err
// 	}
// 	hash := sha3.NewLegacyKeccak256()
// 	hash.Write([]byte{0x19, 0x01})
// 	hash.Write(domainSeparator)
// 	hash.Write(messageHash)
// 	return hash.Sum(nil), nil
// }
