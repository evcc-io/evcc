package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPlanRequiredDurationVehicleRemoved(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicleSoc = 50

	// remove the vehicle while socBasedPlanning reads its capacity, i.e. before the snapshot
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().DoAndReturn(func() float64 {
		lp.vmu.Lock()
		lp.vehicle = nil
		lp.vmu.Unlock()
		return 50
	})
	lp.vehicle = vehicle

	assert.Equal(t, time.Duration(0), lp.GetPlanRequiredDuration(80, 10e3))
}
