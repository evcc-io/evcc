package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// getBatteryMode returns the battery mode
func (site *Site) getBatteryMode() api.BatteryMode {
	site.Lock()
	defer site.Unlock()
	return site.batteryMode
}

// setBatteryMode sets the battery mode
func (site *Site) setBatteryMode(batMode api.BatteryMode) {
	site.Lock()
	defer site.Unlock()
	site.batteryMode = batMode
}

func (site *Site) updateBatteryMode(loadpoints []loadpoint.API) {
	// determine expected state
	batMode := api.BatteryNormal
	for _, lp := range loadpoints {
		if lp.GetStatus() == api.StatusC && (lp.GetMode() == api.ModeNow || lp.GetPlanActive()) {
			batMode = api.BatteryLocked
			break
		}
	}

	if batMode == site.getBatteryMode() {
		return
	}

	// update batteries
	for _, meter := range site.batteryMeters {
		if batCtrl, ok := meter.(api.BatteryController); ok {
			if err := batCtrl.SetBatteryMode(batMode); err != nil {
				site.log.ERROR.Println("battery mode:", err)
				return
			}
		}
	}

	// update state and publish
	site.setBatteryMode(batMode)
	site.publish("batteryMode", batMode)
}
