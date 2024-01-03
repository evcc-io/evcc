package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
)

func batteryModeModified(mode api.BatteryMode) bool {
	return mode != api.BatteryUnknown && mode != api.BatteryNormal
}

// GetBatteryMode returns the battery mode
func (site *Site) GetBatteryMode() api.BatteryMode {
	site.RLock()
	defer site.RUnlock()
	return site.batteryMode
}

// SetBatteryMode sets the battery mode
func (site *Site) SetBatteryMode(batMode api.BatteryMode) {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set battery mode:", batMode)

	if site.batteryMode != batMode {
		site.batteryMode = batMode
		site.publish(keys.BatteryMode, batMode)
	}
}

func (site *Site) determineBatteryMode(loadpoints []loadpoint.API) api.BatteryMode {
	for _, lp := range loadpoints {
		if lp.GetStatus() == api.StatusC && (lp.GetMode() == api.ModeNow || lp.GetPlanActive()) {
			return api.BatteryHold
		}
	}

	return api.BatteryNormal
}

func (site *Site) updateBatteryMode(mode api.BatteryMode) error {
	// update batteries
	for _, meter := range site.batteryMeters {
		if batCtrl, ok := meter.(api.BatteryController); ok {
			if err := batCtrl.SetBatteryMode(mode); err != nil {
				return err
			}
		}
	}

	// update state and publish
	site.SetBatteryMode(mode)

	return nil
}
