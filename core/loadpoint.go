package core

import (
	"fmt"
	"math"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	evbus "github.com/asaskevich/EventBus"
)

const (
	evChargeCurrent = "ChargeCurrent" // update fakeChargeMeter
	evChargePower   = "ChargePower"   // update chargeRater
	evStartCharge   = "StartCharge"   // update chargeTimer
	evStopCharge    = "StopCharge"    // update chargeTimer
)

var status = map[bool]string{false: "disable", true: "enable"}
var presence = map[bool]string{false: "—", true: "✓"}

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power, voltage float64, phases int64) float64 {
	return power / (float64(phases) * voltage)
}

// minInt calculates minimum of two integer values
func minInt(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// LoadPoint is responsible for controlling charge depending on
// SoC needs and power availability.
type LoadPoint struct {
	bus evbus.Bus

	Name        string
	Charger     api.Charger
	ChargeTimer api.ChargeTimer
	ChargeRater api.ChargeRater

	// meters
	GridMeter   api.Meter // Grid usage meter
	PVMeter     api.Meter // PV generation meter
	ChargeMeter api.Meter // Charger usage meter
	SoC         api.SoC   // SoC

	// options
	Phases        int64   // SOC phases. Required for converting power and current.
	MinCurrent    int64   // PV mode: start current	Min+PV mode: min current
	MaxCurrent    int64   // Max allowed current. Physically ensured by the charge controller
	Voltage       float64 // Operating voltage. 230V for Germany.
	ResidualPower float64 // PV meter only: household usage. Grid meter: household safety margin

	state *State // state variables
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint(name string, charger api.Charger) *LoadPoint {
	state := &State{
		mode:          api.ModeNow,
		status:        api.StatusA,
		targetCurrent: -1,
	}

	return &LoadPoint{
		bus:        evbus.New(),
		state:      state,
		Name:       name,
		Phases:     1,
		Voltage:    230, // V
		MinCurrent: 6,   // A
		MaxCurrent: 16,  // A
		Charger:    charger,
	}
}

// Synced returns the synchronized loadpoint state
func (lp *LoadPoint) Synced() *State {
	return lp.state
}

func (lp *LoadPoint) chargeMeterPresent() bool {
	_, isWrapped := lp.ChargeMeter.(*wrapper.ChargeMeter)
	return !isWrapped
}

// Dump loadpoint configuration
func (lp *LoadPoint) Dump() {
	soc := lp.SoC != nil
	grid := lp.GridMeter != nil
	pv := lp.PVMeter != nil
	charge := lp.chargeMeterPresent()
	log.INFO.Printf("%s config: soc %s grid %s pv %s charge %s",
		lp.Name, presence[soc], presence[grid], presence[pv], presence[charge])
	log.INFO.Printf("%s charge mode: %s", lp.Name, lp.state.Mode())
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) Prepare() {
	// ensure charge meter exists
	if lp.ChargeMeter == nil {
		m := &wrapper.ChargeMeter{
			Phases:  lp.Phases,
			Voltage: lp.Voltage,
		}
		_ = lp.bus.Subscribe(evChargeCurrent, m.SetChargeCurrent)
		_ = lp.bus.Subscribe(evStopCharge, func() {
			m.SetChargeCurrent(0)
		})
		lp.ChargeMeter = m
	}

	// ensure charge meter can supply total energy
	if lp.ChargeRater == nil {
		rt := wrapper.NewChargeRater(lp.Name, lp.ChargeMeter)
		_ = lp.bus.Subscribe(evChargePower, rt.SetChargePower)
		_ = lp.bus.Subscribe(evStartCharge, rt.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, rt.StopCharge)
		lp.ChargeRater = rt
	}

	// ensure charge timer exists
	if lp.ChargeTimer == nil {
		if ct, ok := lp.Charger.(api.ChargeTimer); ok {
			lp.ChargeTimer = ct
		} else {
			ct := wrapper.NewChargeTimer()
			_ = lp.bus.Subscribe(evStartCharge, ct.StartCharge)
			_ = lp.bus.Subscribe(evStopCharge, ct.StopCharge)
			lp.ChargeTimer = ct
		}
	}
}

func (lp *LoadPoint) chargeMode(mode api.ChargeMode) {
	log.INFO.Printf("%s set charge mode: %s", lp.Name, string(mode))
	lp.state.SetMode(mode)
}

func (lp *LoadPoint) chargerEnable(enable bool) error {
	log.INFO.Printf("%s charger %s", lp.Name, status[enable])
	return lp.Charger.Enable(enable)
}

// chargingCycle detects charge cycle start and stop events and manages the
// charge energy counter and charge timer. It guards against duplicate invocation.
func (lp *LoadPoint) chargingCycle(enable bool) {
	if enable == lp.state.Charging() {
		return
	}

	if enable {
		log.INFO.Printf("%s start charging ->", lp.Name)
		lp.bus.Publish(evStartCharge)
	} else {
		log.INFO.Printf("%s stop charging <-", lp.Name)
		lp.bus.Publish(evStopCharge)
	}

	lp.state.SetCharging(enable)
}

// updateChargerEnabled checks charger enabled state
func (lp *LoadPoint) updateChargerEnabled() bool {
	enabled, err := lp.Charger.Enabled()
	if err != nil {
		log.ERROR.Printf("%s charger error: %v", lp.Name, err)
		return false
	}
	log.TRACE.Printf("%s charger: %sd", lp.Name, status[enabled])

	if !enabled {
		// stop cycle if running
		lp.chargingCycle(false)
		lp.chargeMode(api.ModeOff)
	}

	return enabled
}

// updateChargeStatus updates car status and stops charging if car disconnected
func (lp *LoadPoint) updateChargeStatus() bool {
	// abort if no vehicle connected
	status, err := lp.Charger.Status()
	if err != nil {
		log.ERROR.Printf("%s charger error: %v", lp.Name, err)
		return false
	}
	log.TRACE.Printf("%s charger car status: %s", lp.Name, status)

	if status != lp.state.Status() {
		lp.state.SetStatus(status)
		if lp.Connected() {
			log.INFO.Printf("%s car connected (%s)", lp.Name, string(status))
		} else {
			log.INFO.Printf("%s car disconnected", lp.Name)
			lp.chargingCycle(false)
		}

		if lp.Charging() {
			lp.chargingCycle(true)
		}
	}

	return lp.Connected()
}

// updateChargeCurrentAndPower retrieves chargers actual current and charge meter power.
// Charger actual current is published and may update the charge power.
// This mechanism ensures that fake charge meter and charge energy are updated.
func (lp *LoadPoint) updateChargeCurrentAndPower() (current int64, power float64, err error) {
	current, err = lp.Charger.ActualCurrent()
	if err == nil {
		log.TRACE.Printf("%s charge current: %dA", lp.Name, current)
	} else {
		return 0, 0, fmt.Errorf("charger error: %v", err)
	}

	power, err = lp.ChargeMeter.CurrentPower()
	if err == nil {
		log.TRACE.Printf("%s charge power: %.0fW", lp.Name, power)
		lp.state.SetChargePower(power)

		// update charge rater with power
		lp.bus.Publish(evChargePower, power)
	} else {
		return 0, 0, fmt.Errorf("charge meter error: %v", err)
	}

	return current, power, nil
}

// setTargetCurrent guards setting current against changing to identical value
// and violating MaxCurrent
func (lp *LoadPoint) setTargetCurrent(chargeCurrent, targetChargeCurrent int64) error {
	if targetChargeCurrent > lp.MaxCurrent {
		targetChargeCurrent = lp.MaxCurrent
		log.WARN.Printf("%s hard limit charge current: %dA", lp.Name, targetChargeCurrent)
	}

	if lp.state.TargetCurrent() != targetChargeCurrent {
		log.INFO.Printf("%s set max charge current: %dA", lp.Name, targetChargeCurrent)
		if err := lp.Charger.(api.ChargeController).MaxCurrent(targetChargeCurrent); err != nil {
			return fmt.Errorf("charge controller error: %v", err)
		}
		lp.state.SetTargetCurrent(targetChargeCurrent)
	}

	// publish in case grid meter not available
	if !lp.Charging() {
		chargeCurrent = 0
	}
	lp.bus.Publish(evChargeCurrent, minInt(chargeCurrent, targetChargeCurrent))

	return nil
}

// Update reevaluates meters and charger state
func (lp *LoadPoint) Update() {
	// check if charging is enabled && car connected
	if enabled := lp.updateChargerEnabled(); !enabled {
		return
	}
	if connected := lp.updateChargeStatus(); !connected {
		return
	}

	// abort if dumb charge controller
	if _, chargeController := lp.Charger.(api.ChargeController); !chargeController {
		log.ERROR.Printf("%s no charge controller assigned", lp.Name)
		return
	}

	// execute loading strategy
	var err error
	mode := lp.state.Mode()
	switch mode {
	case api.ModeNow:
		err = lp.ApplyModeNow()
	case api.ModeMinPV, api.ModePV:
		err = lp.ApplyModePV(mode)
	}

	if err != nil {
		log.ERROR.Printf("%s error: %v", lp.Name, err)
	}
}

// ApplyModeNow updates charging current and sets maximum current
func (lp *LoadPoint) ApplyModeNow() error {
	// get charger current
	chargeCurrent, _, err := lp.updateChargeCurrentAndPower()
	if err != nil {
		return err
	}

	// get max charge current
	targetChargeCurrent := lp.MaxCurrent
	log.DEBUG.Printf("%s target current: %dA (max)", lp.Name, targetChargeCurrent)

	// set max charge current
	return lp.setTargetCurrent(chargeCurrent, targetChargeCurrent)
}

// ApplyModePV sets "minpv" or "pv" load modes
func (lp *LoadPoint) ApplyModePV(mode api.ChargeMode) error {
	// get charger current and charge meter power
	chargeCurrent, chargePower, err := lp.updateChargeCurrentAndPower()
	if err != nil {
		return err
	}

	var targetChargePower float64

	// use grid meter if available, pv meter else
	if lp.GridMeter != nil {
		gridPower, err := lp.GridMeter.CurrentPower()
		if err != nil {
			return fmt.Errorf("grid meter: %v", err)
		}
		log.DEBUG.Printf("%s grid power: %.0fW", lp.Name, gridPower)

		// grid power must be negative for export
		deltaChargePower := -gridPower - lp.ResidualPower
		targetChargePower = chargePower + deltaChargePower

		log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW residual", lp.Name, targetChargePower, chargePower, gridPower, lp.ResidualPower)
	} else {
		pvPower, err := lp.PVMeter.CurrentPower()
		if err != nil {
			return fmt.Errorf("pv meter: %v", err)
		}
		log.DEBUG.Printf("%s pv power: %.0fW", lp.Name, pvPower)

		pvPower = math.Abs(pvPower)
		targetChargePower = pvPower - lp.ResidualPower

		log.DEBUG.Printf("%s target power: %.0fW = %.0fW pv - %.0fW residual", lp.Name, targetChargePower, pvPower, lp.ResidualPower)
	}

	// get max charge current
	targetChargePower = math.Max(targetChargePower, 0)
	targetChargeCurrent := int64(powerToCurrent(targetChargePower, lp.Voltage, lp.Phases))

	if tc := targetChargeCurrent; targetChargeCurrent < lp.MinCurrent {
		switch mode {
		case api.ModeMinPV:
			// charger cannot go below 6A - clamp at min current
			targetChargeCurrent = lp.MinCurrent
		case api.ModePV:
			// If charger is already using less than min current it is probably
			// at the end of the charging cycle. Charge can continue without
			// further limiting the current, otherwise stop charging.
			if chargeCurrent > targetChargeCurrent {
				// disable charger
				targetChargeCurrent = 0
			}
		}
		log.DEBUG.Printf("%s target current: %dA -> %dA override", lp.Name, tc, targetChargeCurrent)
	} else {
		log.DEBUG.Printf("%s target current: %dA", lp.Name, targetChargeCurrent)
	}

	// set max charge current
	return lp.setTargetCurrent(chargeCurrent, targetChargeCurrent)
}
