package circuit

import (
	"errors"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type circuitTest struct {
	// current values for parent, circuit 1, circuit 2
	p, c1, c2 float64
	// old/new demand values and allowed result
	old, new, res float64
}

func circuitTests() []circuitTest {
	return []circuitTest{
		// no load
		{0, 0, 0, 0, 0, 0}, // =
		{0, 0, 0, 0, 1, 1}, // +
		{0, 0, 0, 0, 2, 1}, // +
		{0, 0, 0, 1, 1, 1}, // =

		// circuit 1 loaded
		{0, 1, 0, 0, 0, 0}, // =
		{0, 1, 0, 0, 1, 0}, // +
		{0, 1, 0, 0, 2, 0}, // +
		{0, 1, 0, 1, 1, 1}, // =
		{0, 1, 0, 2, 1, 1}, // -

		// circuit 1 overloaded
		{0, 2, 0, 0, 0, 0}, // =
		{0, 2, 0, 0, 1, 0}, // +
		{0, 2, 0, 1, 1, 0}, // =
		{0, 2, 0, 2, 2, 1}, // =
		{0, 2, 0, 2, 3, 1}, // +
		{0, 2, 0, 2, 1, 1}, // -

		{0, 1.1, 0, 2, 1, 1}, // -
		{0, 1.1, 0, 1, 0, 0}, // -

		// parent loaded
		{1, 0, 0, 0, 0, 0}, // =
		{1, 0, 0, 0, 1, 0}, // +
		{1, 0, 0, 0, 2, 0}, // +
		{1, 0, 0, 1, 1, 1}, // =
		{1, 0, 0, 2, 1, 1}, // -

		// parent overloaded
		{2, 0, 0, 0, 0, 0}, // =
		{2, 0, 0, 0, 1, 0}, // +
		{2, 0, 0, 1, 1, 0}, // =
		{2, 0, 0, 2, 2, 1}, // =
		{2, 0, 0, 2, 3, 1}, // +
		{2, 0, 0, 2, 1, 1}, // -

		{1.1, 0, 0, 2, 1, 1}, // -
		{1.1, 0, 0, 1, 0, 0}, // -

		// negative load
		{-1, -1, 0, 0, 2, 2}, // +
	}
}

func TestCircuitPower(t *testing.T) {
	log := util.NewLogger("foo")

	circ := func(t *testing.T, ctrl *gomock.Controller, maxP float64) (*Circuit, *api.MockMeter) {
		m := api.NewMockMeter(ctrl)
		c, err := New(log, "foo", 0, maxP, m, 0)
		require.NoError(t, err)
		return c, m
	}

	for _, tc := range circuitTests() {
		ctrl := gomock.NewController(t)

		pc, pm := circ(t, ctrl, 1)
		c1, cm1 := circ(t, ctrl, 1)
		c2, cm2 := circ(t, ctrl, 1)

		c1.setParent(pc)
		c2.setParent(pc)

		// update meters
		pm.EXPECT().CurrentPower().Return(tc.p, nil)
		cm1.EXPECT().CurrentPower().Return(tc.c1, nil)
		cm2.EXPECT().CurrentPower().Return(tc.c2, nil)
		require.NoError(t, pc.Update(nil))

		assert.Equal(t, tc.res, c1.ValidatePower(tc.old, tc.new), tc)

		ctrl.Finish()
	}
}

func TestCircuitCurrents(t *testing.T) {
	log := util.NewLogger("foo")

	type combined struct {
		*api.MockMeter
		*api.MockPhaseCurrents
	}
	circ := func(t *testing.T, ctrl *gomock.Controller, maxC float64) (*Circuit, combined) {
		m := combined{
			api.NewMockMeter(ctrl),
			api.NewMockPhaseCurrents(ctrl),
		}
		c, err := New(log, "foo", maxC, 0, m, 0)
		require.NoError(t, err)
		return c, m
	}

	for _, tc := range circuitTests() {
		ctrl := gomock.NewController(t)

		pc, pm := circ(t, ctrl, 1)
		c1, cm1 := circ(t, ctrl, 1)
		c2, cm2 := circ(t, ctrl, 1)

		c1.setParent(pc)
		c2.setParent(pc)

		// update meters
		pm.MockMeter.EXPECT().CurrentPower().AnyTimes().Return(0.0, nil)
		cm1.MockMeter.EXPECT().CurrentPower().AnyTimes().Return(0.0, nil)
		cm2.MockMeter.EXPECT().CurrentPower().AnyTimes().Return(0.0, nil)
		pm.MockPhaseCurrents.EXPECT().Currents().Return(tc.p, tc.p, tc.p, nil)
		cm1.MockPhaseCurrents.EXPECT().Currents().Return(tc.c1, tc.c1, tc.c1, nil)
		cm2.MockPhaseCurrents.EXPECT().Currents().Return(tc.c2, tc.c2, tc.c2, nil)
		require.NoError(t, pc.Update(nil))

		assert.Equal(t, tc.res, c1.ValidateCurrent(tc.old, tc.new), tc)

		ctrl.Finish()
	}
}

// TestHEMSConsumptionClamp verifies that ValidatePower clamps against the
// HEMS consumption limit when one is registered on the root circuit.
func TestHEMSConsumptionClamp(t *testing.T) {
	log := util.NewLogger("foo")

	for _, tc := range []struct {
		name      string
		maxPower  float64
		hemsLimit float64
		request   float64
		want      float64
	}{
		{"no hems, no max", 0, 0, 9000, 9000},
		{"no hems, max set", 5000, 0, 9000, 5000},
		{"hems below max", 5000, 4200, 9000, 4200},
		{"hems above max", 5000, 9000, 9000, 5000},
		{"hems but no user max", 0, 4200, 9000, 4200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(log, "root", 0, tc.maxPower, nil, 0)
			require.NoError(t, err)

			if tc.hemsLimit > 0 {
				ctrl := gomock.NewController(t)
				hems := api.NewMockHEMS(ctrl)
				hems.EXPECT().MaxConsumptionPower().Return(&tc.hemsLimit).AnyTimes()
				c.hems = hems
			}

			assert.Equal(t, tc.want, c.ValidatePower(0, tc.request))
		})
	}
}

type phaseMeter struct {
	*api.MockMeter
	*api.MockPhaseCurrents
}

func newPhaseMeter(ctrl *gomock.Controller, power float64, currents [3]float64) phaseMeter {
	m := phaseMeter{api.NewMockMeter(ctrl), api.NewMockPhaseCurrents(ctrl)}
	m.MockMeter.EXPECT().CurrentPower().Return(power, nil).AnyTimes()
	m.MockPhaseCurrents.EXPECT().Currents().Return(currents[0], currents[1], currents[2], nil).AnyTimes()
	return m
}

// TestSelfLimitingBatteries verifies that loadpoints may use the power of a battery that
// keeps grid import under the limit itself, but never more than the circuit limit.
func TestSelfLimitingBatteries(t *testing.T) {
	log := util.NewLogger("foo")

	for _, tc := range []struct {
		name        string
		power       float64 // measured power
		currents    [3]float64
		batPower    float64 // negative is charging
		batCurrents *[3]float64
		resP        float64 // allowed power from 0 to 11000W
		resC        float64 // allowed current from 0 to 16A
	}{
		{"no battery", 10000, [3]float64{14, 14, 14}, 0, nil, 1000, 2},
		{"charging", 10000, [3]float64{14, 14, 14}, -9000, &[3]float64{13, 13, 13}, 10000, 15},
		{"charging partly from pv", 2000, [3]float64{3, 3, 3}, -9000, &[3]float64{13, 13, 13}, 11000, 16},
		{"charging, unknown currents", 10000, [3]float64{14, 14, 14}, -9000, nil, 10000, 2},
		{"charging 1p", 7000, [3]float64{14, 4, 4}, -3000, &[3]float64{13, 0, 0}, 7000, 12},
		{"discharging", 10000, [3]float64{14, 14, 14}, 3000, &[3]float64{4, 4, 4}, 1000, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			c, err := New(log, "root", 16, 11000, newPhaseMeter(ctrl, tc.power, tc.currents), 0)
			require.NoError(t, err)

			if tc.name != "no battery" {
				var bat api.Meter
				if tc.batCurrents != nil {
					bat = newPhaseMeter(ctrl, tc.batPower, *tc.batCurrents)
				} else {
					m := api.NewMockMeter(ctrl)
					m.EXPECT().CurrentPower().Return(tc.batPower, nil)
					bat = m
				}
				c.batteryMeters = []api.Meter{bat}
			}

			require.NoError(t, c.Update(nil))

			assert.Equal(t, tc.resP, c.ValidatePower(0, 11000), "power")
			assert.Equal(t, tc.resC, c.ValidateCurrent(0, 16), "current")

			// published values remain measured values
			assert.Equal(t, tc.power, c.GetChargePower())
		})
	}
}

// TestSelfLimitingBatteriesError verifies that an unreadable battery counts as fixed load.
func TestSelfLimitingBatteriesError(t *testing.T) {
	log := util.NewLogger("foo")
	ctrl := gomock.NewController(t)

	c, err := New(log, "root", 16, 11000, newPhaseMeter(ctrl, 10000, [3]float64{14, 14, 14}), 0)
	require.NoError(t, err)

	charging := newPhaseMeter(ctrl, -4000, [3]float64{6, 6, 6})
	broken := api.NewMockMeter(ctrl)
	broken.EXPECT().CurrentPower().Return(0.0, backoff.Permanent(errors.New("timeout")))
	c.batteryMeters = []api.Meter{charging, broken}

	require.NoError(t, c.Update(nil))

	assert.Equal(t, 1000.0, c.ValidatePower(0, 11000))
	assert.Equal(t, 2.0, c.ValidateCurrent(0, 16))
}

// TestSelfLimitingBatteriesChildLimit verifies that a child circuit keeps its own limit.
func TestSelfLimitingBatteriesChildLimit(t *testing.T) {
	log := util.NewLogger("foo")
	ctrl := gomock.NewController(t)

	pm := api.NewMockMeter(ctrl)
	cm := api.NewMockMeter(ctrl)
	bat := api.NewMockMeter(ctrl)
	pm.EXPECT().CurrentPower().Return(10000.0, nil)
	cm.EXPECT().CurrentPower().Return(9000.0, nil)
	bat.EXPECT().CurrentPower().Return(-9000.0, nil)

	parent, err := New(log, "root", 0, 11000, pm, 0)
	require.NoError(t, err)
	parent.batteryMeters = []api.Meter{bat}

	child, err := New(log, "child", 0, 11000, cm, 0)
	require.NoError(t, err)
	require.NoError(t, child.setParent(parent))

	require.NoError(t, parent.Update(nil))

	// the battery frees the parent, the child limit remains
	assert.Equal(t, 2000.0, child.ValidatePower(0, 11000))
	assert.Equal(t, 10000.0, parent.ValidatePower(0, 11000))
}

// TestSelfLimitingBatteriesBuffer verifies that the buffer keeps part of the battery
// share as load, but never more than the measured load.
func TestSelfLimitingBatteriesBuffer(t *testing.T) {
	log := util.NewLogger("foo")

	for _, tc := range []struct {
		name       string
		batPower   float64
		batCurrent float64
		resP, resC float64 // allowed power from 0 to 11000W, current from 0 to 16A
	}{
		{"buffer below battery share", -9000, 13, 8620, 13},
		{"buffer above battery share", -1000, 1, 1000, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			c, err := New(log, "root", 16, 11000, newPhaseMeter(ctrl, 10000, [3]float64{14, 14, 14}), 0)
			require.NoError(t, err)
			c.batteryMeters = []api.Meter{newPhaseMeter(ctrl, tc.batPower, [3]float64{tc.batCurrent, tc.batCurrent, tc.batCurrent})}
			c.batteryBuffer = 2

			require.NoError(t, c.Update(nil))

			assert.Equal(t, tc.resP, c.ValidatePower(0, 11000), "power")
			assert.Equal(t, tc.resC, c.ValidateCurrent(0, 16), "current")
		})
	}
}
