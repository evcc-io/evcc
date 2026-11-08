package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestPlanRequiredDurationVehicleRemoved(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicleSoc = 50

	// SoC goal obtained while the vehicle was present, must not be read as energy
	assert.Equal(t, time.Duration(0), lp.GetPlanRequiredDuration(80, 10e3, true))
}
