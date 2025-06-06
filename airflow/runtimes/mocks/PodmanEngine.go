// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	spinner "github.com/briandowns/spinner"

	types "github.com/astronomer/astro-cli/airflow/runtimes/types"
)

// PodmanEngine is an autogenerated mock type for the PodmanEngine type
type PodmanEngine struct {
	mock.Mock
}

// InitializeMachine provides a mock function with given fields: name, s
func (_m *PodmanEngine) InitializeMachine(name string, s *spinner.Spinner) error {
	ret := _m.Called(name, s)

	if len(ret) == 0 {
		panic("no return value specified for InitializeMachine")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *spinner.Spinner) error); ok {
		r0 = rf(name, s)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InspectMachine provides a mock function with given fields: name
func (_m *PodmanEngine) InspectMachine(name string) (*types.InspectedMachine, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for InspectMachine")
	}

	var r0 *types.InspectedMachine
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*types.InspectedMachine, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *types.InspectedMachine); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.InspectedMachine)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListContainers provides a mock function with no fields
func (_m *PodmanEngine) ListContainers() ([]types.ListedContainer, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ListContainers")
	}

	var r0 []types.ListedContainer
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.ListedContainer, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.ListedContainer); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.ListedContainer)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListMachines provides a mock function with no fields
func (_m *PodmanEngine) ListMachines() ([]types.ListedMachine, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ListMachines")
	}

	var r0 []types.ListedMachine
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.ListedMachine, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.ListedMachine); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.ListedMachine)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveMachine provides a mock function with given fields: name
func (_m *PodmanEngine) RemoveMachine(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for RemoveMachine")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetMachineAsDefault provides a mock function with given fields: name
func (_m *PodmanEngine) SetMachineAsDefault(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for SetMachineAsDefault")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartMachine provides a mock function with given fields: name
func (_m *PodmanEngine) StartMachine(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for StartMachine")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopMachine provides a mock function with given fields: name
func (_m *PodmanEngine) StopMachine(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for StopMachine")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewPodmanEngine creates a new instance of PodmanEngine. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPodmanEngine(t interface {
	mock.TestingT
	Cleanup(func())
}) *PodmanEngine {
	mock := &PodmanEngine{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
