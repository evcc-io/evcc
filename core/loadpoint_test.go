package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/push"
	"github.com/golang/mock/gomock"
)

func TestNew(t *testing.T) {
	lp := NewLoadPoint()

	if lp.Mode != api.ModeOff {
		t.Errorf("Mode %v", lp.Mode)
	}
	if lp.Phases != 1 {
		t.Errorf("Phases %v", lp.Phases)
	}
	if lp.MinCurrent != 6 {
		t.Errorf("MinCurrent %v", lp.MinCurrent)
	}
	if lp.MaxCurrent != 16 {
		t.Errorf("MaxCurrent %v", lp.MaxCurrent)
	}
	if lp.Steepness != 10 {
		t.Errorf("Steepness %v", lp.Steepness)
	}
	if lp.status != api.StatusNone {
		t.Errorf("status %v", lp.status)
	}
	if lp.enabled {
		t.Errorf("enabled %v", lp.enabled)
	}
	if lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
	if lp.targetCurrent != 0 {
		t.Errorf("targetCurrent %v", lp.targetCurrent)
	}
}

func newLoadPoint(charger api.Charger, pv, grid, charge api.Meter) *LoadPoint {
	lp := NewLoadPoint()

	lp.Charger = charger
	lp.PVMeter = pv
	lp.GridMeter = grid
	lp.ChargeMeter = charge

	uiChan := make(chan Param)
	notificationChan := make(chan push.Event)

	lp.Prepare(uiChan, notificationChan)

	go func() {
		for {
			select {
			case <-uiChan:
			case <-notificationChan:
			}
		}
	}()

	return lp
}

func newEnvironment(t *testing.T, ctrl *gomock.Controller, chargeMeter bool) (
	*LoadPoint, *mock.MockCharger,
	*mock.MockMeter, *mock.MockMeter, *mock.MockMeter,
) {
	wb := mock.NewMockCharger(ctrl)
	pm := mock.NewMockMeter(ctrl)
	gm := mock.NewMockMeter(ctrl)
	cm := mock.NewMockMeter(ctrl)

	if !chargeMeter {
		cm = nil
	}

	wb.EXPECT().Enabled().Return(true, nil) // initial alignment with wb

	lp := newLoadPoint(wb, pm, gm, cm)

	if !lp.enabled {
		t.Errorf("enabled %v", lp.enabled)
	}

	pm.EXPECT().CurrentPower().Return(float64(1e3), nil)
	gm.EXPECT().CurrentPower().Return(float64(1e3), nil)
	cm.EXPECT().CurrentPower().Return(float64(0), nil)

	return lp, wb, pm, gm, cm
}

func TestStatusA(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lp, wb, _, _, _ := newEnvironment(t, ctrl, true)

	wb.EXPECT().Status().Return(api.StatusA, nil)
	lp.update()

	if lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
	if lp.targetCurrent != 0 {
		t.Errorf("targetCurrent %v", lp.targetCurrent)
	}
}

func TestStatusB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lp, wb, _, _, _ := newEnvironment(t, ctrl, true)
	lp.enabled = false

	wb.EXPECT().Status().Return(api.StatusB, nil)
	lp.update()

	if lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
	if lp.targetCurrent != 0 {
		t.Errorf("targetCurrent %v", lp.targetCurrent)
	}
}

func TestStatusC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lp, wb, _, _, _ := newEnvironment(t, ctrl, true)
	lp.enabled = false

	wb.EXPECT().Status().Return(api.StatusC, nil)
	lp.update()

	if !lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
	if lp.targetCurrent != 0 {
		t.Errorf("targetCurrent %v", lp.targetCurrent)
	}
}
