// Code generated by mockery v2.37.1. DO NOT EDIT.

package secp256k1mocks

import (
	secp256k1 "github.com/hyperledger/firefly-signer/pkg/secp256k1"
	mock "github.com/stretchr/testify/mock"
)

// Signer is an autogenerated mock type for the Signer type
type Signer struct {
	mock.Mock
}

// Sign provides a mock function with given fields: msgToHashAndSign
func (_m *Signer) Sign(msgToHashAndSign []byte) (*secp256k1.SignatureData, error) {
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

// NewSigner creates a new instance of Signer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSigner(t interface {
	mock.TestingT
	Cleanup(func())
}) *Signer {
	mock := &Signer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
