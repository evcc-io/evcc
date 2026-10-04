package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestBatteryModesPerDevice guards that in automatic mode each battery follows its own
// optimizer suggestion instead of the mode of the first battery
func TestBatteryModesPerDevice(t *testing.T) {
	enableAutomatic(t)
	ctrl := gomock.NewController(t)

	active, activeCon := batteryControlMock(ctrl, 50, 100)
	idle, idleCon := batteryControlMock(ctrl, 50, 100)

	site := &Site{
		log: util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "active"}, active),
			config.NewStaticDevice(config.Named{Name: "idle"}, idle),
		},
		suggestions: map[string]types.Suggestion{
			batteryKey("active"): {Action: api.BatteryNormal.String()},
			batteryKey("idle"):   {Action: api.BatteryHold.String()},
		},
	}

	activeCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
	idleCon.EXPECT().SetBatteryMode(api.BatteryHold).Times(1)

	for range 3 {
		site.updateBatteryMode(false, false, api.Rate{})
	}

	assert.Equal(t, api.BatteryNormal, site.batteryMode, "site mode follows the first battery")
}
