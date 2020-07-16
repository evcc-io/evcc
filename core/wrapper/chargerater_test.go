package wrapper

import (
	"testing"
	"time"

	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/util"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

func TestNoMeter(t *testing.T) {
	cr := NewChargeRater(util.NewLogger("foo"), nil)
	clck := clock.NewMock()
	cr.clck = clck

	cr.StartCharge()

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

	f, err := cr.ChargedEnergy()

	if f != 1 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}
}
func TestWrappedMeter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mm := mock.NewMockMeter(ctrl)
	me := mock.NewMockMeterEnergy(ctrl)
	cm := &meter.MeterEnergyDecorator{Meter: mm, MeterEnergy: me}

	me.EXPECT().
		TotalEnergy().
		Return(2.0, nil)
	me.EXPECT().
		TotalEnergy().
		Return(3.0, nil)

	cr := NewChargeRater(util.NewLogger("foo"), cm)
	clck := clock.NewMock()
	cr.clck = clck

	cr.StartCharge()

	// ignored with meter present
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)
	clck.Add(time.Hour)
	cr.SetChargePower(0)

	cr.StopCharge()

	// ignored with meter present
	clck.Add(time.Hour)
	cr.SetChargePower(1e3)

	f, err := cr.ChargedEnergy()

	if f != 1 || err != nil {
		t.Errorf("energy: %.1f %v", f, err)
	}
}
