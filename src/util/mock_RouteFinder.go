// Code generated by mockery v2.9.6. DO NOT EDIT.

package util

import (
	mock "github.com/stretchr/testify/mock"
	netlink "github.com/vishvananda/netlink"
)

// MockRouteFinder is an autogenerated mock type for the RouteFinder type
type MockRouteFinder struct {
	mock.Mock
}

// LinkByName provides a mock function with given fields: name
func (_m *MockRouteFinder) LinkByName(name string) (netlink.Link, error) {
	ret := _m.Called(name)

	var r0 netlink.Link
	if rf, ok := ret.Get(0).(func(string) netlink.Link); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(netlink.Link)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RouteList provides a mock function with given fields: link, family
func (_m *MockRouteFinder) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	ret := _m.Called(link, family)

	var r0 []netlink.Route
	if rf, ok := ret.Get(0).(func(netlink.Link, int) []netlink.Route); ok {
		r0 = rf(link, family)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]netlink.Route)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(netlink.Link, int) error); ok {
		r1 = rf(link, family)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockRouteFinder interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockRouteFinder creates a new instance of MockRouteFinder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRouteFinder(t mockConstructorTestingTNewMockRouteFinder) *MockRouteFinder {
	mock := &MockRouteFinder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
