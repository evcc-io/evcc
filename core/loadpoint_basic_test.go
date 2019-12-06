package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/mock"
	"github.com/golang/mock/gomock"
)

func TestChargerEnableNoChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Enabled().
		Return(false, nil)

	lp := NewLoadPoint("lp1", c)
	if err := lp.chargerEnable(false); err != nil {
		t.Error(err)
	}
}

func TestChargerEnableChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Enabled().
		Return(false, nil)
	c.EXPECT().
		Enable(true).
		Return(nil)

	lp := NewLoadPoint("lp1", c)
	if err := lp.chargerEnable(true); err != nil {
		t.Error(err)
	}
}

func TestEVNotConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Status().
		Return(api.StatusA, nil)

	lp := NewLoadPoint("lp1", c)

	if res := lp.updateChargeStatus(); res {
		t.Error("ev not disconnected")
	}
}
func TestChargerEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Enabled().
		Return(true, nil)

	lp := NewLoadPoint("lp1", c)

	if enabled := lp.updateChargerEnabled(); !enabled {
		t.Error("charger not enabled")
	}
}
func TestChargerDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Enabled().
		Return(false, nil)

	lp := NewLoadPoint("lp1", c)
	lp.state.SetMode(api.ModeNow)

	if enabled := lp.updateChargerEnabled(); enabled {
		t.Error("charger not disabled")
	}
}

func TestEVConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	c.EXPECT().
		Status().
		Return(api.StatusC, nil)

	lp := NewLoadPoint("lp1", c)

	if res := lp.updateChargeStatus(); !res {
		t.Error("ev not connected")
	}
}

func TestStartCharging(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	mm := mock.NewMockMeter(ctrl)
	me := mock.NewMockMeterEnergy(ctrl)
	me.EXPECT().
		TotalEnergy().
		Return(0.0, nil)

	lp := NewLoadPoint("lp1", c)
	lp.state.SetCharging(false)
	lp.ChargeMeter = &wrapper.CompositeMeter{Meter: mm, MeterEnergy: me}

	lp.chargingCycle(true)
}
func TestStopCharging(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := mock.NewMockCharger(ctrl)
	mm := mock.NewMockMeter(ctrl)
	me := mock.NewMockMeterEnergy(ctrl)
	me.EXPECT().
		TotalEnergy().
		Return(10.0, nil)

	lp := NewLoadPoint("lp1", c)
	lp.state.SetCharging(true)
	lp.ChargeMeter = &wrapper.CompositeMeter{Meter: mm, MeterEnergy: me}

	lp.chargingCycle(false)
}
