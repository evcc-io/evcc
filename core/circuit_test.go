package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCircuitPower(t *testing.T) {
	log := util.NewLogger("foo")

	circ := func(t *testing.T, ctrl *gomock.Controller, maxP float64) (*Circuit, *api.MockMeter) {
		m := api.NewMockMeter(ctrl)
		c, err := NewCircuit(log, "foo", 0, maxP, m)
		require.NoError(t, err)
		return c, m
	}

	for _, tc := range []struct {
		pm, cm1, cm2 float64
		req, res     float64
	}{
		// no load
		{0, 0, 0, 0, 0},
		{0, 0, 0, 1, 1},
		{0, 0, 0, 2, 1},

		// c1 loaded
		{0, 1, 0, 0, 0},
		{0, 1, 0, 1, 0},
		{0, 1, 0, 2, 0},

		// pc loaded
		{1, 0, 0, 0, 0},
		{1, 0, 0, 1, 0},
		{1, 0, 0, 2, 0},
	} {
		ctrl := gomock.NewController(t)

		pc, pm := circ(t, ctrl, 1)
		c1, cm1 := circ(t, ctrl, 1)
		c2, cm2 := circ(t, ctrl, 1)

		c1.SetParent(pc)
		c2.SetParent(pc)

		// update meters
		pm.EXPECT().CurrentPower().Return(tc.pm, nil)
		cm1.EXPECT().CurrentPower().Return(tc.cm1, nil)
		cm2.EXPECT().CurrentPower().Return(tc.cm2, nil)
		require.NoError(t, pc.Update(nil))

		require.Equal(t, tc.res, c1.ValidatePower(0, tc.req))

		ctrl.Finish()
	}
}

// func TestCircuitCurrents(t *testing.T) {
// 	log := util.NewLogger("foo")

// 	type mockMeter struct {
// 		*api.MockMeter
// 		*api.MockPhaseCurrents
// 	}

// 	circ := func(t *testing.T, ctrl *gomock.Controller, maxP float64) (*Circuit, *mockMeter) {
// 		m := api.NewMockMeter(ctrl)
// 		mc := api.NewMockPhaseCurrents(ctrl)
// 		mm := &mockMeter{m, mc}
// 		c, err := NewCircuit(log, 0, maxP, mm)
// 		require.NoError(t, err)
// 		return c, mm
// 	}

// 	for _, tc := range []struct {
// 		pm, cm1, cm2 float64
// 		req, res     float64
// 	}{
// 		// no load
// 		{0, 0, 0, 0, 0},
// 		{0, 0, 0, 1, 1},
// 		{0, 0, 0, 2, 1},

// 		// c1 loaded
// 		{0, 1, 0, 0, 0},
// 		{0, 1, 0, 1, 0},
// 		{0, 1, 0, 2, 0},

// 		// pc loaded
// 		{1, 0, 0, 0, 0},
// 		{1, 0, 0, 1, 0},
// 		{1, 0, 0, 2, 0},
// 	} {
// 		ctrl := gomock.NewController(t)

// 		pc, pm := circ(t, ctrl, 1)
// 		c1, cm1 := circ(t, ctrl, 1)
// 		c2, cm2 := circ(t, ctrl, 1)

// 		c1.SetParent(pc)
// 		c2.SetParent(pc)

// 		// update meters
// 		pm.MockMeter.EXPECT().CurrentPower().Return(tc.pm, nil)
// 		cm1.MockMeter.EXPECT().CurrentPower().Return(tc.cm1, nil)
// 		cm2.MockMeter.EXPECT().CurrentPower().Return(tc.cm2, nil)

// 		// update meters
// 		pm.MockPhaseCurrents.EXPECT().Currents().Return(tc.pm, tc.pm, tc.pm, nil)
// 		cm1.MockPhaseCurrents.EXPECT().Currents().Return(tc.cm1, tc.cm1, tc.cm1, nil)
// 		cm2.MockPhaseCurrents.EXPECT().Currents().Return(tc.cm2, tc.cm2, tc.cm2, nil)
// 		require.NoError(t, pc.Update(nil))

// 		require.Equal(t, tc.res, c1.ValidatePower(0, tc.req))
// 		require.Equal(t, tc.res, c1.ValidateCurrent(0, tc.req))

// 		ctrl.Finish()
// 	}
// }
