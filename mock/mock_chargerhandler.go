// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/andig/evcc/core (interfaces: Handler)

// Package mock is a generated GoMock package.
package mock

import (
	api "github.com/andig/evcc/api"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockHandler is a mock of Handler interface
type MockHandler struct {
	ctrl     *gomock.Controller
	recorder *MockHandlerMockRecorder
}

// MockHandlerMockRecorder is the mock recorder for MockHandler
type MockHandlerMockRecorder struct {
	mock *MockHandler
}

// NewMockHandler creates a new mock instance
func NewMockHandler(ctrl *gomock.Controller) *MockHandler {
	mock := &MockHandler{ctrl: ctrl}
	mock.recorder = &MockHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockHandler) EXPECT() *MockHandlerMockRecorder {
	return m.recorder
}

// Enabled mocks base method
func (m *MockHandler) Enabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enabled indicates an expected call of Enabled
func (mr *MockHandlerMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*MockHandler)(nil).Enabled))
}

// Prepare mocks base method
func (m *MockHandler) Prepare() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Prepare")
}

// Prepare indicates an expected call of Prepare
func (mr *MockHandlerMockRecorder) Prepare() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Prepare", reflect.TypeOf((*MockHandler)(nil).Prepare))
}

// Ramp mocks base method
func (m *MockHandler) Ramp(arg0 int64, arg1 ...bool) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Ramp", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ramp indicates an expected call of Ramp
func (mr *MockHandlerMockRecorder) Ramp(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ramp", reflect.TypeOf((*MockHandler)(nil).Ramp), varargs...)
}

// Status mocks base method
func (m *MockHandler) Status() (api.ChargeStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(api.ChargeStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Status indicates an expected call of Status
func (mr *MockHandlerMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockHandler)(nil).Status))
}

// Sync mocks base method
func (m *MockHandler) Sync() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Sync")
}

// Sync indicates an expected call of Sync
func (mr *MockHandlerMockRecorder) Sync() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sync", reflect.TypeOf((*MockHandler)(nil).Sync))
}

// TargetCurrent mocks base method
func (m *MockHandler) TargetCurrent() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TargetCurrent")
	ret0, _ := ret[0].(int64)
	return ret0
}

// TargetCurrent indicates an expected call of TargetCurrent
func (mr *MockHandlerMockRecorder) TargetCurrent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TargetCurrent", reflect.TypeOf((*MockHandler)(nil).TargetCurrent))
}
