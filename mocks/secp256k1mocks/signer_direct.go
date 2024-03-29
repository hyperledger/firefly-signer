// Code generated by mockery v2.37.1. DO NOT EDIT.

package secp256k1mocks

import (
	secp256k1 "github.com/hyperledger/firefly-signer/pkg/secp256k1"
	mock "github.com/stretchr/testify/mock"
)

// SignerDirect is an autogenerated mock type for the SignerDirect type
type SignerDirect struct {
	mock.Mock
}

// Sign provides a mock function with given fields: msgToHashAndSign
func (_m *SignerDirect) Sign(msgToHashAndSign []byte) (*secp256k1.SignatureData, error) {
	ret := _m.Called(msgToHashAndSign)

	var r0 *secp256k1.SignatureData
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) (*secp256k1.SignatureData, error)); ok {
		return rf(msgToHashAndSign)
	}
	if rf, ok := ret.Get(0).(func([]byte) *secp256k1.SignatureData); ok {
		r0 = rf(msgToHashAndSign)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*secp256k1.SignatureData)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(msgToHashAndSign)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SignDirect provides a mock function with given fields: message
func (_m *SignerDirect) SignDirect(message []byte) (*secp256k1.SignatureData, error) {
	ret := _m.Called(message)

	var r0 *secp256k1.SignatureData
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) (*secp256k1.SignatureData, error)); ok {
		return rf(message)
	}
	if rf, ok := ret.Get(0).(func([]byte) *secp256k1.SignatureData); ok {
		r0 = rf(message)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*secp256k1.SignatureData)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(message)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewSignerDirect creates a new instance of SignerDirect. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSignerDirect(t interface {
	mock.TestingT
	Cleanup(func())
}) *SignerDirect {
	mock := &SignerDirect{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
