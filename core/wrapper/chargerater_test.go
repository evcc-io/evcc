package wrapper

import (
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
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

	mm := api.NewMockMeter(ctrl)
	me := api.NewMockMeterEnergy(ctrl)

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

// TestDeferredBaseline covers the OCPP transaction-recovery case: the meter is
// not yet readable when StartCharge fires, so the baseline must be latched on
// the first successful TotalEnergy() read instead of defaulting to zero
// (which would cause the lifetime register to be reported as session energy).
func TestDeferredBaseline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mm := api.NewMockMeter(ctrl)
	me := api.NewMockMeterEnergy(ctrl)

	type EnergyDecorator struct {
		api.Meter
		api.MeterEnergy
	}

	cm := &EnergyDecorator{Meter: mm, MeterEnergy: me}

	cr := NewChargeRater(util.NewLogger("foo"), cm)
	clck := clock.NewMock()
	cr.clck = clck

	// meter not yet available at StartCharge — recovered transaction before first MeterValues
	me.EXPECT().TotalEnergy().Return(0.0, errors.New("not available"))

	cr.StartCharge(true)

	// first read also fails — must surface the error, not a bogus delta
	me.EXPECT().TotalEnergy().Return(0.0, errors.New("not available"))

	if _, err := cr.ChargedEnergy(); err == nil {
		t.Errorf("expected error while meter unavailable")
	}

	// first successful read latches the baseline (lifetime register, e.g. 939 kWh)
	me.EXPECT().TotalEnergy().Return(939.080, nil)

	if f, err := cr.ChargedEnergy(); f != 0 || err != nil {
		t.Errorf("expected 0 on baseline-latch read, got %.3f %v", f, err)
	}

	// subsequent reads return delta against the latched baseline
	me.EXPECT().TotalEnergy().Return(942.080, nil)

	if f, err := cr.ChargedEnergy(); f != 3 || err != nil {
		t.Errorf("expected 3kWh delta, got %.3f %v", f, err)
	}

	me.EXPECT().TotalEnergy().Return(944.080, nil)

	cr.StopCharge()

	if f, err := cr.ChargedEnergy(); f != 5 || err != nil {
		t.Errorf("final energy: %.1f %v", f, err)
	}
}

func TestExtendCharge(t *testing.T) {
	ctrl := gomock.NewController(t)

	total := 100.0
	m := api.NewMockMeter(ctrl)
	me := struct {
		api.Meter
		api.MeterEnergy
	}{m, &stubMeterEnergy{total: &total}}

	cr := NewChargeRater(util.NewLogger("foo"), me)

	cr.StartCharge(true)
	total = 110
	cr.StopCharge()

	// final reading reported by the charger after stop
	cr.ExtendCharge(110.07)
	if f, _ := cr.ChargedEnergy(); f < 10.069 || f > 10.071 {
		t.Errorf("energy: %.3f", f)
	}

	// repeated or lower readings are ignored
	cr.ExtendCharge(110.07)
	cr.ExtendCharge(109)
	if f, _ := cr.ChargedEnergy(); f < 10.069 || f > 10.071 {
		t.Errorf("energy: %.3f", f)
	}

	// implausible jump
	cr.ExtendCharge(150)
	if f, _ := cr.ChargedEnergy(); f < 10.069 || f > 10.071 {
		t.Errorf("energy: %.3f", f)
	}

	// no effect while charging
	cr.StartCharge(true)
	cr.ExtendCharge(111)
	if f, _ := cr.ChargedEnergy(); f < 10.069 || f > 10.071 {
		t.Errorf("energy: %.3f", f)
	}
}

type stubMeterEnergy struct{ total *float64 }

func (s *stubMeterEnergy) TotalEnergy() (float64, error) { return *s.total, nil }
