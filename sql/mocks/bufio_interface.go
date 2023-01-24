// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	bufio "bufio"
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// BufIOBind is an autogenerated mock type for the BufIOBind type
type BufIOBind struct {
	mock.Mock
}

// NewScanner provides a mock function with given fields: r
func (_m *BufIOBind) NewScanner(r io.Reader) *bufio.Scanner {
	ret := _m.Called(r)

	var r0 *bufio.Scanner
	if rf, ok := ret.Get(0).(func(io.Reader) *bufio.Scanner); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bufio.Scanner)
		}
	}

	return r0
}

type mockConstructorTestingTNewBufIOBind interface {
	mock.TestingT
	Cleanup(func())
}

// NewBufIOBind creates a new instance of BufIOBind. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBufIOBind(t mockConstructorTestingTNewBufIOBind) *BufIOBind {
	mock := &BufIOBind{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}