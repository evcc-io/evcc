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

// stubLoad is a CircuitLoad with directly settable arbitration inputs
type stubLoad struct {
	circuit                api.Circuit
	rank                   int
	power, current         float64
	demandPower, demandCur float64
}

func (l *stubLoad) GetChargePower() float64               { return l.power }
func (l *stubLoad) GetMaxPhaseCurrent() float64           { return l.current }
func (l *stubLoad) GetCircuit() api.Circuit               { return l.circuit }
func (l *stubLoad) GetRank() int                          { return l.rank }
func (l *stubLoad) GetDeadlineDemand() (float64, float64) { return l.demandPower, l.demandCur }

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

		assert.Equal(t, tc.res, c1.ValidatePower(&stubLoad{circuit: c1}, tc.old, tc.new), tc)

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

		assert.Equal(t, tc.res, c1.ValidateCurrent(&stubLoad{circuit: c1}, tc.old, tc.new), tc)

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

			assert.Equal(t, tc.want, c.ValidatePower(&stubLoad{circuit: c}, 0, tc.request))
		})
	}
}

// TestRankReservation verifies that circuit capacity is arbitrated by rank instead
// of by whoever asks first: a lower-ranked load must leave room for the unmet
// deadline demand of a higher-ranked load on the same circuit.
func TestRankReservation(t *testing.T) {
	log := util.NewLogger("foo")

	c, err := New(log, "root", 16, 10000, nil, 0)
	require.NoError(t, err)

	low := &stubLoad{circuit: c, rank: 0}
	high := &stubLoad{circuit: c, rank: rankForced}
	require.NoError(t, c.Update([]api.CircuitLoad{low, high}))

	// no deadline anywhere- the low-ranked load may use the whole circuit
	assert.Equal(t, 10000.0, c.ValidatePower(low, 0, 10000))
	assert.Equal(t, 16.0, c.ValidateCurrent(low, 0, 16))

	// the high-ranked load needs 6000W / 9A more than it draws
	high.demandPower, high.demandCur = 6000, 9

	assert.Equal(t, 4000.0, c.ValidatePower(low, 0, 10000), "low rank must leave the reservation free")
	assert.Equal(t, 7.0, c.ValidateCurrent(low, 0, 16), "low rank must leave the reservation free")

	// the high-ranked load itself never reserves against itself
	assert.Equal(t, 10000.0, c.ValidatePower(high, 0, 10000))
	assert.Equal(t, 16.0, c.ValidateCurrent(high, 0, 16))

	// the reservation stacks with load already measured on the circuit
	low.power, low.current = 2000, 4
	require.NoError(t, c.Update([]api.CircuitLoad{low, high}))
	assert.Equal(t, 2000.0, c.ValidatePower(low, 0, 10000), "measured load and reservation both apply")
	assert.Equal(t, 3.0, c.ValidateCurrent(low, 0, 16), "measured load and reservation both apply")

	// equal rank does not reserve, otherwise both would starve each other
	low.rank = rankForced
	assert.Equal(t, 8000.0, c.ValidatePower(low, 0, 10000))
}

// TestRankReservationParent verifies the reservation also applies on a parent
// circuit, where the competing loads hang off different children.
func TestRankReservationParent(t *testing.T) {
	log := util.NewLogger("foo")

	pc, err := New(log, "root", 0, 10000, nil, 0)
	require.NoError(t, err)
	c1, err := New(log, "c1", 0, 10000, nil, 0)
	require.NoError(t, err)
	c2, err := New(log, "c2", 0, 10000, nil, 0)
	require.NoError(t, err)
	require.NoError(t, c1.setParent(pc))
	require.NoError(t, c2.setParent(pc))

	low := &stubLoad{circuit: c1, rank: 0}
	high := &stubLoad{circuit: c2, rank: rankPlanActive, demandPower: 6000}
	require.NoError(t, pc.Update([]api.CircuitLoad{low, high}))

	// c1 alone has room, but the shared parent must hold 6000W for the plan
	assert.Equal(t, 4000.0, c1.ValidatePower(low, 0, 10000))
}

// rank tiers mirroring core.rankPlanActive/rankForced, kept local to avoid an
// import cycle between core and core/circuit
const (
	rankPlanActive = 1000
	rankForced     = 2000
)
