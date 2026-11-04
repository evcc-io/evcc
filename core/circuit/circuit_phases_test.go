package circuit

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testLoad struct {
	circuit  api.Circuit
	currents [3]float64
	mapped   bool
}

func (l *testLoad) GetChargePower() float64 { return 0 }
func (l *testLoad) GetMaxPhaseCurrent() float64 {
	return max(l.currents[0], l.currents[1], l.currents[2])
}
func (l *testLoad) GetGridPhaseCurrents() [3]float64 { return l.currents }
func (l *testLoad) GetCircuit() api.Circuit          { return l.circuit }
func (l *testLoad) HasGridPhaseMapping() bool        { return l.mapped }

func TestCircuitGridPhases(t *testing.T) {
	l1 := api.GridPhases{true, false, false}
	l3 := api.GridPhases{false, false, true}

	for _, tc := range []struct {
		name     string
		currents [3]float64 // existing load
		phases   api.GridPhases
		old, new float64
		res      float64
	}{
		// unknown phases are reported on all grid phases, as before
		{"unknown 1p, 1p on unknown", [3]float64{20, 20, 20}, api.AllGridPhases, 0, 16, 0},
		{"unknown 1p, 1p on L3", [3]float64{20, 20, 20}, l3, 0, 16, 0},
		{"mapped 1p L1, 1p on unknown", [3]float64{20, 0, 0}, api.AllGridPhases, 0, 16, 0},
		{"mapped 1p L1, 1p on L1", [3]float64{14, 0, 0}, l1, 0, 16, 6},
		{"mapped 1p L1, 1p on L3", [3]float64{20, 0, 0}, l3, 0, 16, 16},
		{"mapped 1p L1, 1p on L3 increase", [3]float64{20, 0, 16}, l3, 16, 20, 20},
		{"mapped 1p L1, 3p", [3]float64{14, 0, 0}, api.AllGridPhases, 0, 10, 6},
		{"mapped 3p, 1p on L3", [3]float64{10, 10, 10}, l3, 0, 16, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(util.NewLogger("foo"), "foo", 20, 0, nil, 0)
			require.NoError(t, err)

			require.NoError(t, c.Update([]api.CircuitLoad{&testLoad{circuit: c, currents: tc.currents}}))
			assert.Equal(t, tc.res, c.ValidateCurrent(tc.phases, tc.old, tc.new))
		})
	}
}

func TestCircuitGridPhasesParent(t *testing.T) {
	log := util.NewLogger("foo")

	parent, err := New(log, "parent", 35, 0, nil, 0)
	require.NoError(t, err)
	child, err := New(log, "child", 20, 0, nil, 0)
	require.NoError(t, err)
	require.NoError(t, child.setParent(parent))

	loads := []api.CircuitLoad{
		&testLoad{circuit: child, currents: [3]float64{20, 0, 0}, mapped: true},
		&testLoad{circuit: parent, currents: [3]float64{0, 0, 30}, mapped: true},
	}
	require.NoError(t, parent.Update(loads))

	assert.Equal(t, [3]float64{20, 0, 30}, parent.GetGridPhaseCurrents())
	assert.Equal(t, 30.0, parent.GetMaxPhaseCurrent())

	// child has room on L3, parent limits L3 to 5A
	assert.Equal(t, 5.0, child.ValidateCurrent(api.GridPhases{false, false, true}, 0, 16))
	// L2 is free on both
	assert.Equal(t, 16.0, child.ValidateCurrent(api.GridPhases{false, true, false}, 0, 16))
}

func TestCircuitPhaseMappingMismatch(t *testing.T) {
	log := util.NewLogger("foo")

	type combined struct {
		*api.MockMeter
		*api.MockPhaseCurrents
	}

	for _, tc := range []struct {
		name     string
		root     bool
		measured [3]float64
		mismatch int
	}{
		{"match", false, [3]float64{0, 0, 16}, 0},
		{"match with other load", false, [3]float64{5, 0, 17}, 0},
		{"mismatch", false, [3]float64{16, 0, 0}, mismatchCycles},
		{"root ignored", true, [3]float64{16, 0, 0}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := combined{api.NewMockMeter(ctrl), api.NewMockPhaseCurrents(ctrl)}
			m.MockMeter.EXPECT().CurrentPower().Return(0.0, nil).AnyTimes()
			m.MockPhaseCurrents.EXPECT().Currents().Return(tc.measured[0], tc.measured[1], tc.measured[2], nil).AnyTimes()

			c, err := New(log, "foo", 20, 0, m, 0)
			require.NoError(t, err)

			if !tc.root {
				parent, err := New(log, "parent", 0, 0, nil, 0)
				require.NoError(t, err)
				require.NoError(t, c.setParent(parent))
			}

			loads := []api.CircuitLoad{
				&testLoad{circuit: c, currents: [3]float64{0, 0, 16}, mapped: true},
				&testLoad{circuit: c, currents: [3]float64{4, 4, 4}},
			}

			for range mismatchCycles {
				require.NoError(t, c.Update(loads))
			}

			assert.Equal(t, tc.mismatch, c.mappingMismatch)
		})
	}
}
