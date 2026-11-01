package core

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util/config"
)

const (
	// batterySlaveOutageTimeout is how long slaves keep their last setpoint without readings.
	batterySlaveOutageTimeout = 3 * time.Minute
	// Ignore direction differences within this 50 W deadband.
	batterySlavePowerDirectionTolerance = 50.0
	// Reapply mode only after opposite-direction measurements persist this long.
	batterySlaveModeMismatchTimeout = 30 * time.Second
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

type batterySlaveState struct {
	appliedMode            api.BatteryMode
	appliedSetpoint        *batterySlaveSetpoint
	directionMismatchSince time.Time
}

type batterySlavePowerMismatch uint8

const (
	batterySlavePowerMatches batterySlavePowerMismatch = iota
	batterySlavePowerOppositeDirection
)

func batterySlaveSetpointMismatch(applied batterySlaveSetpoint, measuredPower float64) batterySlavePowerMismatch {
	if applied.power <= batterySlavePowerDirectionTolerance {
		return batterySlavePowerMatches
	}

	switch applied.mode {
	case api.BatteryCharge:
		if measuredPower > batterySlavePowerDirectionTolerance {
			return batterySlavePowerOppositeDirection
		}
	case api.BatteryDischarge:
		if measuredPower < -batterySlavePowerDirectionTolerance {
			return batterySlavePowerOppositeDirection
		}
	}

	return batterySlavePowerMatches
}

func (site *Site) batterySlaveState(name string) *batterySlaveState {
	if site.batterySlaveStates == nil {
		site.batterySlaveStates = make(map[string]*batterySlaveState)
	}

	state := site.batterySlaveStates[name]
	if state == nil {
		state = new(batterySlaveState)
		site.batterySlaveStates[name] = state
	}
	return state
}

func (site *Site) batterySlaveModeMismatchExpired(state *batterySlaveState, desired batterySlaveSetpoint, measuredPower float64) bool {
	if state.appliedSetpoint == nil ||
		state.appliedMode != desired.mode ||
		state.appliedSetpoint.mode != desired.mode ||
		desired.power <= batterySlavePowerDirectionTolerance {
		state.directionMismatchSince = time.Time{}
		return false
	}

	mismatch := batterySlaveSetpointMismatch(*state.appliedSetpoint, measuredPower)
	if mismatch == batterySlavePowerMatches {
		state.directionMismatchSince = time.Time{}
		return false
	}

	now := site.clock.Now()
	if state.directionMismatchSince.IsZero() {
		state.directionMismatchSince = now
		return false
	}
	if now.Sub(state.directionMismatchSince) < batterySlaveModeMismatchTimeout {
		return false
	}

	state.directionMismatchSince = now
	return true
}

func (site *Site) resetBatterySlaveDirectionMismatch(slaves []int) {
	for _, slaveIndex := range slaves {
		name := site.batteryMeters[slaveIndex].Config().Name
		if state := site.batterySlaveStates[name]; state != nil {
			state.directionMismatchSince = time.Time{}
		}
	}
}

func (site *Site) setBatterySlaveMode(state *batterySlaveState, meter api.Meter, mode api.BatteryMode) error {
	controller, _ := api.Cap[api.BatteryController](meter)
	if err := controller.SetBatteryMode(mode); err != nil {
		return err
	}

	if state.appliedMode != mode {
		state.directionMismatchSince = time.Time{}
	}
	state.appliedMode = mode
	return nil
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

// releaseBatterySlave returns a slave under power control to normal operation.
func (site *Site) releaseBatterySlave(dev config.Device[api.Meter]) bool {
	name := dev.Config().Name
	state := site.batterySlaveStates[name]
	if state == nil || !batteryModeModified(state.appliedMode) {
		return false
	}

	ctrl, ok := api.Cap[api.BatteryController](dev.Instance())
	if !ok {
		return false
	}

	if err := ctrl.SetBatteryMode(api.BatteryNormal); err != nil {
		site.log.ERROR.Printf("battery %s mode: %v", deviceTitleOrName(dev), err)
		return false
	}

	delete(site.batterySlaveStates, name)
	return true
}

// releaseBatterySlaves returns all slaves under power control to normal operation.
func (site *Site) releaseBatterySlaves() {
	for _, dev := range site.batteryMeters {
		site.releaseBatterySlave(dev)
	}
}

func (site *Site) updateBatterySlavePower(mm []types.Measurement, powerValid []bool) {
	if len(mm) != len(site.batteryMeters) || len(powerValid) != len(mm) {
		site.log.ERROR.Printf("battery slave power: inconsistent input lengths: %d measurements, %d power-valid flags, %d batteries", len(mm), len(powerValid), len(site.batteryMeters))
		return
	}
	if site.batterySlaveOutageSince == nil {
		site.batterySlaveOutageSince = make(map[string]time.Time)
	}

	markSlavesFailed := func(slaves []int) {
		for _, slaveIndex := range slaves {
			if mm[slaveIndex].PowerControlStatus != "overridden" {
				mm[slaveIndex].PowerControlStatus = "failed"
			}
		}
	}

	// slaves keep their last setpoint during an outage until the timeout releases them
	handleOutage := func(master string, slaves []int) {
		markSlavesFailed(slaves)
		site.resetBatterySlaveDirectionMismatch(slaves)

		now := site.clock.Now()
		since, ok := site.batterySlaveOutageSince[master]
		if !ok {
			site.batterySlaveOutageSince[master] = now
			return
		}
		if now.Sub(since) < batterySlaveOutageTimeout {
			return
		}

		for _, slaveIndex := range slaves {
			slave := site.batteryMeters[slaveIndex]
			if site.releaseBatterySlave(slave) {
				site.log.WARN.Printf("battery slave %s: released to normal mode, no readings for %s", deviceTitleOrName(slave), batterySlaveOutageTimeout)
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
		masterName := dev.Config().Name
		slaves := slavesByMaster[masterName]
		if len(slaves) == 0 {
			continue
		}
		for _, slaveIndex := range slaves {
			slave := site.batteryMeters[slaveIndex]
			name := slave.Config().Name
			if !site.batterySlaveModeAllowed(name) {
				delete(site.batterySlaveStates, name)
				mm[slaveIndex].PowerControlStatus = "overridden"
			}
		}

		masterTitle := deviceTitleOrName(dev)
		if !powerValid[masterIndex] {
			site.log.WARN.Printf("battery master %s: cannot set slave power, master power unavailable", masterTitle)
			handleOutage(masterName, slaves)
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
			handleOutage(masterName, slaves)
			continue
		}

		plan, valid := calculateBatterySlaveSetpoints(masterPower, slavePowers)
		if !valid {
			site.log.WARN.Printf("battery master %s: cannot set slave power, invalid total %.0fW", masterTitle, plan.totalPower)
			handleOutage(masterName, slaves)
			continue
		}
		delete(site.batterySlaveOutageSince, masterName)
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
			state := site.batterySlaveState(name)
			reapplyMode := site.batterySlaveModeMismatchExpired(state, setpoint, mm[slaveIndex].Power)
			if state.appliedMode != setpoint.mode || reapplyMode {
				if err := site.setBatterySlaveMode(state, meter, setpoint.mode); err != nil {
					site.log.ERROR.Printf("battery %s mode: %v", slaveTitle, err)
					mm[slaveIndex].PowerControlStatus = "failed"
					continue
				}

				if reapplyMode {
					site.log.WARN.Printf("battery slave %s: measured power %.0fW contradicts last setpoint, reapplying mode %s", slaveTitle, mm[slaveIndex].Power, setpoint.mode)
				}
			}

			controller, _ := api.Cap[api.BatteryModePowerController](meter)
			if err := controller.SetBatteryModePower(setpoint.mode, setpoint.power); err != nil {
				site.log.ERROR.Printf("battery %s power setpoint: %v", slaveTitle, err)
				mm[slaveIndex].PowerControlStatus = "failed"
			} else {
				state.appliedSetpoint = &setpoint
				site.log.TRACE.Printf("battery slave %s: applied target %s %.0fW", slaveTitle, setpoint.mode, setpoint.power)
				mm[slaveIndex].PowerControlStatus = "applied"
			}
		}
	}
}
