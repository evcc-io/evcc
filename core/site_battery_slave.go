package core

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
)

// SupportsBatterySlaveControl reports whether a meter supports the modes and setpoint control required for battery slave control.
func SupportsBatterySlaveControl(meter api.Meter) bool {
	controller, ok := api.Cap[api.BatteryController](meter)
	if !ok {
		return false
	}

	modes := controller.BatteryModes()
	return slices.Contains(modes, api.BatteryCharge) &&
		slices.Contains(modes, api.BatteryDischarge) &&
		api.HasCap[api.BatteryModePowerController](meter)
}

func (site *Site) batterySlaveModeAllowed(name string) bool {
	switch site.batteryModeApplied[name] {
	case api.BatteryUnknown, api.BatteryNormal:
		return true
	default:
		return false
	}
}

type batterySlaveSetpoint struct {
	mode  api.BatteryMode
	power float64
}

type batterySlavePowerPlan struct {
	totalPower   float64
	averagePower float64
	setpoints    []batterySlaveSetpoint
}

// calculateBatterySlaveSetpoints splits evenly by count, ignoring capacity, soc and power limits.
func calculateBatterySlaveSetpoints(masterPower float64, slavePowers []float64) (batterySlavePowerPlan, bool) {
	plan := batterySlavePowerPlan{totalPower: masterPower}
	if len(slavePowers) == 0 {
		return plan, false
	}

	for _, power := range slavePowers {
		plan.totalPower += power
	}
	if math.IsNaN(plan.totalPower) || math.IsInf(plan.totalPower, 0) {
		return plan, false
	}

	plan.averagePower = plan.totalPower / float64(len(slavePowers)+1)

	mode := api.BatteryDischarge
	if plan.averagePower < 0 {
		mode = api.BatteryCharge
	}

	setpoint := math.Abs(plan.averagePower)
	plan.setpoints = make([]batterySlaveSetpoint, len(slavePowers))
	for i := range plan.setpoints {
		plan.setpoints[i] = batterySlaveSetpoint{mode: mode, power: setpoint}
	}

	return plan, true
}

// releaseBatterySlaves returns slaves under power control to normal operation.
func (site *Site) releaseBatterySlaves() {
	for _, dev := range site.batteryMeters {
		name := dev.Config().Name
		if !batteryModeModified(site.batterySlaveModeApplied[name]) {
			continue
		}

		ctrl, ok := api.Cap[api.BatteryController](dev.Instance())
		if !ok {
			continue
		}

		if err := ctrl.SetBatteryMode(api.BatteryNormal); err != nil {
			site.log.ERROR.Printf("battery %s mode: %v", deviceTitleOrName(dev), err)
			continue
		}
		delete(site.batterySlaveModeApplied, name)
	}
}

func (site *Site) updateBatterySlavePower(mm []types.Measurement, powerValid []bool) {
	if len(mm) != len(site.batteryMeters) || len(powerValid) != len(mm) {
		site.log.ERROR.Printf("battery slave power: inconsistent input lengths: %d measurements, %d power-valid flags, %d batteries", len(mm), len(powerValid), len(site.batteryMeters))
		return
	}
	if site.batterySlaveModeApplied == nil {
		site.batterySlaveModeApplied = make(map[string]api.BatteryMode)
	}

	markSlavesFailed := func(slaves []int) {
		for _, slaveIndex := range slaves {
			if mm[slaveIndex].PowerControlStatus != "overridden" {
				mm[slaveIndex].PowerControlStatus = "failed"
			}
		}
	}

	slavesByMaster := make(map[string][]int)
	for i, dev := range site.batteryMeters {
		if master, ok := dev.Config().Property("master").(string); ok && master != "" {
			slavesByMaster[master] = append(slavesByMaster[master], i)
		}
	}

	for masterIndex, dev := range site.batteryMeters {
		slaves := slavesByMaster[dev.Config().Name]
		if len(slaves) == 0 {
			continue
		}
		for _, slaveIndex := range slaves {
			slave := site.batteryMeters[slaveIndex]
			name := slave.Config().Name
			if !site.batterySlaveModeAllowed(name) {
				delete(site.batterySlaveModeApplied, name)
				mm[slaveIndex].PowerControlStatus = "overridden"
			}
		}

		masterTitle := deviceTitleOrName(dev)
		if !powerValid[masterIndex] {
			site.log.WARN.Printf("battery master %s: cannot set slave power, master power unavailable", masterTitle)
			markSlavesFailed(slaves)
			continue
		}

		masterPower := mm[masterIndex].Power
		slavePowers := make([]float64, 0, len(slaves))
		readingsValid := true
		powerParts := []string{fmt.Sprintf("%s=%.0fW", masterTitle, masterPower)}
		for _, slaveIndex := range slaves {
			if !powerValid[slaveIndex] {
				slaveTitle := deviceTitleOrName(site.batteryMeters[slaveIndex])
				site.log.WARN.Printf("battery master %s: cannot set slave power, slave %s power unavailable", masterTitle, slaveTitle)
				readingsValid = false
				break
			}
			slavePowers = append(slavePowers, mm[slaveIndex].Power)
			powerParts = append(powerParts, fmt.Sprintf("%s=%.0fW", deviceTitleOrName(site.batteryMeters[slaveIndex]), mm[slaveIndex].Power))
		}
		if !readingsValid {
			markSlavesFailed(slaves)
			continue
		}

		plan, valid := calculateBatterySlaveSetpoints(masterPower, slavePowers)
		if !valid {
			site.log.WARN.Printf("battery master %s: cannot set slave power, invalid total %.0fW", masterTitle, plan.totalPower)
			markSlavesFailed(slaves)
			continue
		}
		site.log.TRACE.Printf("battery master %s: [%s] total %.0fW, average %.0fW across %d slaves", masterTitle, strings.Join(powerParts, ", "), plan.totalPower, plan.averagePower, len(plan.setpoints))
		for i, slaveIndex := range slaves {
			setpoint := plan.setpoints[i]
			slave := site.batteryMeters[slaveIndex]
			slaveTitle := deviceTitleOrName(slave)
			if mm[slaveIndex].PowerControlStatus == "overridden" {
				site.log.TRACE.Printf("battery slave %s: skipping power control, global battery mode is active", slaveTitle)
				continue
			}

			meter := slave.Instance()
			if !SupportsBatterySlaveControl(meter) {
				site.log.WARN.Printf("battery slave %s: cannot set power, charge/discharge power control is not supported", slaveTitle)
				mm[slaveIndex].PowerControlStatus = "failed"
				continue
			}

			name := slave.Config().Name
			if site.batterySlaveModeApplied[name] != setpoint.mode {
				modeController, _ := api.Cap[api.BatteryController](meter)
				if err := modeController.SetBatteryMode(setpoint.mode); err != nil {
					site.log.ERROR.Printf("battery %s mode: %v", slaveTitle, err)
					mm[slaveIndex].PowerControlStatus = "failed"
					continue
				}

				site.batterySlaveModeApplied[name] = setpoint.mode
			}

			controller, _ := api.Cap[api.BatteryModePowerController](meter)
			if err := controller.SetBatteryModePower(setpoint.mode, setpoint.power); err != nil {
				site.log.ERROR.Printf("battery %s power setpoint: %v", slaveTitle, err)
				mm[slaveIndex].PowerControlStatus = "failed"
			} else {
				site.log.TRACE.Printf("battery slave %s: applied target %s %.0fW", slaveTitle, setpoint.mode, setpoint.power)
				mm[slaveIndex].PowerControlStatus = "applied"
			}
		}
	}
}
