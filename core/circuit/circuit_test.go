package circuit

import (
	"testing"

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

// TestBatteryLimitsGridImport verifies that battery grid charging only yields to
// managed loadpoints and only if the user opted in.
func TestBatteryLimitsGridImport(t *testing.T) {
	log := util.NewLogger("foo")

	type combined struct {
		*api.MockMeter
		*api.MockPhaseCurrents
	}

	for _, tc := range []struct {
		name                 string
		optIn                bool
		power                float64 // measured power
		currents             [3]float64
		batPower             float64
		batCurrents          [3]float64
		managedP, unmanagedP float64 // allowed power from 0 to 11000W
		managedC, unmanagedC float64 // allowed current from 0 to 16A
	}{
		{"no opt-in", false, 10000, [3]float64{14, 14, 14}, 9000, [3]float64{13, 13, 13}, 1000, 1000, 2, 2},
		{"opt-in", true, 10000, [3]float64{14, 14, 14}, 9000, [3]float64{13, 13, 13}, 10000, 1000, 15, 2},
		{"opt-in, battery partly from pv", true, 2000, [3]float64{3, 3, 3}, 9000, [3]float64{13, 13, 13}, 11000, 9000, 16, 13},
		{"opt-in, unknown battery currents", true, 10000, [3]float64{14, 14, 14}, 9000, [3]float64{}, 10000, 1000, 2, 2},
		{"opt-in, 1p battery", true, 7000, [3]float64{14, 4, 4}, 3000, [3]float64{13, 0, 0}, 7000, 4000, 12, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			m := combined{api.NewMockMeter(ctrl), api.NewMockPhaseCurrents(ctrl)}
			m.MockMeter.EXPECT().CurrentPower().Return(tc.power, nil)
			m.MockPhaseCurrents.EXPECT().Currents().Return(tc.currents[0], tc.currents[1], tc.currents[2], nil)

			c, err := New(log, "root", 16, 11000, m, 0)
			require.NoError(t, err)
			c.batteryLimitsGridImport = tc.optIn
			require.NoError(t, c.Update(nil))

			c.SetBatteryCharge(tc.batPower, tc.batCurrents[0], tc.batCurrents[1], tc.batCurrents[2])

			assert.Equal(t, tc.managedP, c.ValidateManagedPower(0, 11000), "managed power")
			assert.Equal(t, tc.unmanagedP, c.ValidatePower(0, 11000), "unmanaged power")
			assert.Equal(t, tc.managedC, c.ValidateManagedCurrent(0, 16), "managed current")
			assert.Equal(t, tc.unmanagedC, c.ValidateCurrent(0, 16), "unmanaged current")
		})
	}
}

// TestBatteryLimitsGridImportRootOnly verifies that a child circuit does not free battery power.
func TestBatteryLimitsGridImportRootOnly(t *testing.T) {
	log := util.NewLogger("foo")
	ctrl := gomock.NewController(t)

	pm := api.NewMockMeter(ctrl)
	cm := api.NewMockMeter(ctrl)
	pm.EXPECT().CurrentPower().Return(10000.0, nil)
	cm.EXPECT().CurrentPower().Return(9000.0, nil)

	parent, err := New(log, "root", 0, 11000, pm, 0)
	require.NoError(t, err)
	parent.batteryLimitsGridImport = true

	child, err := New(log, "child", 0, 11000, cm, 0)
	require.NoError(t, err)
	require.NoError(t, child.setParent(parent))

	require.NoError(t, parent.Update(nil))

	// battery set on child is ignored, the parent without battery charge limits
	child.SetBatteryCharge(9000, 0, 0, 0)
	assert.Equal(t, 1000.0, child.ValidateManagedPower(0, 11000))

	// battery on root frees the parent, the child limit remains
	parent.SetBatteryCharge(9000, 0, 0, 0)
	assert.Equal(t, 2000.0, child.ValidateManagedPower(0, 11000))
}
