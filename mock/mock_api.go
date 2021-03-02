// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mark-sch/evcc/api (interfaces: Charger,Meter,MeterEnergy,Vehicle,ChargeRater)

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	api "github.com/mark-sch/evcc/api"
	reflect "reflect"
)

// MockCharger is a mock of Charger interface
type MockCharger struct {
	ctrl     *gomock.Controller
	recorder *MockChargerMockRecorder
}

// MockChargerMockRecorder is the mock recorder for MockCharger
type MockChargerMockRecorder struct {
	mock *MockCharger
}

// NewMockCharger creates a new mock instance
func NewMockCharger(ctrl *gomock.Controller) *MockCharger {
	mock := &MockCharger{ctrl: ctrl}
	mock.recorder = &MockChargerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCharger) EXPECT() *MockChargerMockRecorder {
	return m.recorder
}

// Enable mocks base method
func (m *MockCharger) Enable(arg0 bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enable", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Enable indicates an expected call of Enable
func (mr *MockChargerMockRecorder) Enable(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enable", reflect.TypeOf((*MockCharger)(nil).Enable), arg0)
}

// Enabled mocks base method
func (m *MockCharger) Enabled() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Enabled indicates an expected call of Enabled
func (mr *MockChargerMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*MockCharger)(nil).Enabled))
}

// MaxCurrent mocks base method
func (m *MockCharger) MaxCurrent(arg0 int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaxCurrent", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// MaxCurrent indicates an expected call of MaxCurrent
func (mr *MockChargerMockRecorder) MaxCurrent(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaxCurrent", reflect.TypeOf((*MockCharger)(nil).MaxCurrent), arg0)
}

// Status mocks base method
func (m *MockCharger) Status() (api.ChargeStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(api.ChargeStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Status indicates an expected call of Status
func (mr *MockChargerMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockCharger)(nil).Status))
}

// MockMeter is a mock of Meter interface
type MockMeter struct {
	ctrl     *gomock.Controller
	recorder *MockMeterMockRecorder
}

// MockMeterMockRecorder is the mock recorder for MockMeter
type MockMeterMockRecorder struct {
	mock *MockMeter
}

// NewMockMeter creates a new mock instance
func NewMockMeter(ctrl *gomock.Controller) *MockMeter {
	mock := &MockMeter{ctrl: ctrl}
	mock.recorder = &MockMeterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMeter) EXPECT() *MockMeterMockRecorder {
	return m.recorder
}

// CurrentPower mocks base method
func (m *MockMeter) CurrentPower() (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CurrentPower")
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CurrentPower indicates an expected call of CurrentPower
func (mr *MockMeterMockRecorder) CurrentPower() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CurrentPower", reflect.TypeOf((*MockMeter)(nil).CurrentPower))
}

// MockMeterEnergy is a mock of MeterEnergy interface
type MockMeterEnergy struct {
	ctrl     *gomock.Controller
	recorder *MockMeterEnergyMockRecorder
}

// MockMeterEnergyMockRecorder is the mock recorder for MockMeterEnergy
type MockMeterEnergyMockRecorder struct {
	mock *MockMeterEnergy
}

// NewMockMeterEnergy creates a new mock instance
func NewMockMeterEnergy(ctrl *gomock.Controller) *MockMeterEnergy {
	mock := &MockMeterEnergy{ctrl: ctrl}
	mock.recorder = &MockMeterEnergyMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMeterEnergy) EXPECT() *MockMeterEnergyMockRecorder {
	return m.recorder
}

// TotalEnergy mocks base method
func (m *MockMeterEnergy) TotalEnergy() (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TotalEnergy")
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TotalEnergy indicates an expected call of TotalEnergy
func (mr *MockMeterEnergyMockRecorder) TotalEnergy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TotalEnergy", reflect.TypeOf((*MockMeterEnergy)(nil).TotalEnergy))
}

// MockVehicle is a mock of Vehicle interface
type MockVehicle struct {
	ctrl     *gomock.Controller
	recorder *MockVehicleMockRecorder
}

// MockVehicleMockRecorder is the mock recorder for MockVehicle
type MockVehicleMockRecorder struct {
	mock *MockVehicle
}

// NewMockVehicle creates a new mock instance
func NewMockVehicle(ctrl *gomock.Controller) *MockVehicle {
	mock := &MockVehicle{ctrl: ctrl}
	mock.recorder = &MockVehicleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockVehicle) EXPECT() *MockVehicleMockRecorder {
	return m.recorder
}

// Capacity mocks base method
func (m *MockVehicle) Capacity() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Capacity")
	ret0, _ := ret[0].(int64)
	return ret0
}

// Capacity indicates an expected call of Capacity
func (mr *MockVehicleMockRecorder) Capacity() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Capacity", reflect.TypeOf((*MockVehicle)(nil).Capacity))
}

// SoC mocks base method
func (m *MockVehicle) SoC() (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SoC")
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SoC indicates an expected call of SoC
func (mr *MockVehicleMockRecorder) SoC() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SoC", reflect.TypeOf((*MockVehicle)(nil).SoC))
}

// Title mocks base method
func (m *MockVehicle) Title() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Title")
	ret0, _ := ret[0].(string)
	return ret0
}

// Title indicates an expected call of Title
func (mr *MockVehicleMockRecorder) Title() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Title", reflect.TypeOf((*MockVehicle)(nil).Title))
}

// MockChargeRater is a mock of ChargeRater interface
type MockChargeRater struct {
	ctrl     *gomock.Controller
	recorder *MockChargeRaterMockRecorder
}

// MockChargeRaterMockRecorder is the mock recorder for MockChargeRater
type MockChargeRaterMockRecorder struct {
	mock *MockChargeRater
}

// NewMockChargeRater creates a new mock instance
func NewMockChargeRater(ctrl *gomock.Controller) *MockChargeRater {
	mock := &MockChargeRater{ctrl: ctrl}
	mock.recorder = &MockChargeRaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockChargeRater) EXPECT() *MockChargeRaterMockRecorder {
	return m.recorder
}

// ChargedEnergy mocks base method
func (m *MockChargeRater) ChargedEnergy() (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChargedEnergy")
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChargedEnergy indicates an expected call of ChargedEnergy
func (mr *MockChargeRaterMockRecorder) ChargedEnergy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChargedEnergy", reflect.TypeOf((*MockChargeRater)(nil).ChargedEnergy))
}
