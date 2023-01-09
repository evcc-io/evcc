package wrapper

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestNoMeter(t *testing.T) {
	cr := NewChargeRater(util.NewLogger("foo"), nil)
	clck := clock.NewMock()
	cr.clck = clck

	cr.StartCharge(false)
	clck.Add(time.Hour)

	if f, err := cr.ChargedEnergy(); f != 0 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}

	cr.StartCharge(true)

	// 1kWh
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)
	cr.SetChargePower(1e3) // should be ignored as time is identical

	// 0kWh
	clck.Add(time.Hour)
	cr.SetChargePower(0)

	cr.StopCharge()

	// 1kWh - not counted
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)

	if f, err := cr.ChargedEnergy(); f != 1 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}

	// continue
	cr.StartCharge(true)

	// 1kWh
	clck.Add(2 * time.Hour)
	cr.SetChargePower(1e3)
	cr.StopCharge()

	if f, err := cr.ChargedEnergy(); f != 3 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}
}

func TestWrappedMeter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mm := mock.NewMockMeter(ctrl)
	me := mock.NewMockMeterEnergy(ctrl)

	type EnergyDecorator struct {
		api.Meter
		api.MeterEnergy
	}

	cm := &EnergyDecorator{Meter: mm, MeterEnergy: me}

	cr := NewChargeRater(util.NewLogger("foo"), cm)
	clck := clock.NewMock()
	cr.clck = clck

	me.EXPECT().TotalEnergy().Return(2.0, nil)

	cr.StartCharge(false)

	// ignored with meter present
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)
	clck.Add(time.Hour)
	cr.SetChargePower(0)

	me.EXPECT().TotalEnergy().Return(3.0, nil)

	cr.StopCharge()

	if f, err := cr.ChargedEnergy(); f != 1 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}

	// ignored with meter present
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)

	// continue
	me.EXPECT().TotalEnergy().Return(10.0, nil)

	cr.StartCharge(true)
	clck.Add(time.Hour) // actual timing ignored as energy comes from meter

	me.EXPECT().TotalEnergy().Return(12.0, nil)

	cr.StopCharge()

	if f, err := cr.ChargedEnergy(); f != 3 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}
}
