package core

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/push"

	evbus "github.com/asaskevich/EventBus"
)

const (
	evChargeCurrent = "ChargeCurrent" // update fakeChargeMeter
	evChargePower   = "ChargePower"   // update chargeRater
	evStartCharge   = "StartCharge"   // update chargeTimer
	evStopCharge    = "StopCharge"    // update chargeTimer
)

var (
	once     sync.Once
	status   = map[bool]string{false: "disable", true: "enable"}
	presence = map[bool]string{false: "—", true: "✓"}
)

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power, voltage float64, phases int64) int64 {
	return int64(power / (float64(phases) * voltage))
}

// LoadPoint is responsible for controlling charge depending on
// SoC needs and power availability.
type LoadPoint struct {
	bus        evbus.Bus         // event bus
	updateChan chan<- push.Event // notifications

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
	Steepness     int64   // Step size of current change
	Phases        int64   // SOC phases. Required for converting power and current.
	MinCurrent    int64   // PV mode: start current	Min+PV mode: min current
	MaxCurrent    int64   // Max allowed current. Physically ensured by the charge controller
	Voltage       float64 // Operating voltage. 230V for Germany.
	ResidualPower float64 // PV meter only: household usage. Grid meter: household safety margin

	cycleUpdated  time.Time     // charger enabled/disabled timestamp
	CycleDuration time.Duration // charger enable/disable minimum holding time

	state *State // state variables
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint(name string, charger api.Charger) *LoadPoint {
	state := &State{
		mode:          api.ModeNow,
		status:        api.StatusA,
		targetCurrent: 0,
	}

	return &LoadPoint{
		bus:           evbus.New(),
		state:         state,
		Name:          name,
		Phases:        1,
		Voltage:       230, // V
		MinCurrent:    6,   // A
		MaxCurrent:    16,  // A
		Steepness:     1,   // A
		Charger:       charger,
		CycleDuration: time.Minute,
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

func (lp *LoadPoint) handleChargeStart() {
	lp.updateChan <- push.Event{
		EventId: push.ChargeStart,
		Sender:  "Wallbe",
		Attributes: map[string]interface{}{
			"lp":   lp.Name,
			"mode": lp.state.Mode(),
		},
	}
}

func (lp *LoadPoint) handleChargeStop() {
	energy, err := lp.ChargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s %v", lp.Name, err)
	}

	lp.updateChan <- push.Event{
		EventId: push.ChargeStop,
		Sender:  "Wallbe",
		Attributes: map[string]interface{}{
			"lp":     lp.Name,
			"energy": energy,
		},
	}
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) prepare(updateChan chan<- push.Event) {
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

	// event handlers
	lp.updateChan = updateChan
	lp.bus.Subscribe(evStartCharge, lp.handleChargeStart)
	lp.bus.Subscribe(evStopCharge, lp.handleChargeStop)
}

func (lp *LoadPoint) chargeMode(mode api.ChargeMode) {
	log.INFO.Printf("%s set charge mode: %s", lp.Name, string(mode))
	lp.state.SetMode(mode)
}

// chargerEnable switches charging on or off. Minimum cycle duration is guaranteed.
func (lp *LoadPoint) chargerEnable(enable bool) error {
	if lp.state.TargetCurrent() != 0 && lp.state.TargetCurrent() != lp.MinCurrent {
		log.FATAL.Fatal("charger enable/disable called without setting min current first - aborting")
	}

	if time.Since(lp.cycleUpdated) < lp.CycleDuration {
		log.TRACE.Printf("%s ignore charger %s", lp.Name, status[enable])
		return nil
	}

	log.INFO.Printf("%s charger %s", lp.Name, status[enable])

	err := lp.Charger.Enable(enable)
	if err == nil {
		lp.cycleUpdated = time.Now()
	}
	return err
}

// updateChargerEnabled checks charger enabled state
func (lp *LoadPoint) updateChargerEnabled() bool {
	enabled, err := lp.Charger.Enabled()
	// log.DEBUG.Printf(". enabled %v", enabled)

	if err != nil {
		log.ERROR.Printf("%s charger error: %v", lp.Name, err)
		return false
	}
	log.TRACE.Printf("%s charger: %sd", lp.Name, status[enabled])

	return enabled
}

// chargingCycle detects charge cycle start and stop events and manages the
// charge energy counter and charge timer. It guards against duplicate invocation.
func (lp *LoadPoint) chargingCycle(enable bool) {
	if enable == lp.state.Charging() {
		return
	}

	lp.state.SetCharging(enable)

	if enable {
		log.INFO.Printf("%s start charging ->", lp.Name)
		lp.bus.Publish(evStartCharge)
	} else {
		log.INFO.Printf("%s stop charging <-", lp.Name)
		lp.bus.Publish(evStopCharge)
	}
}

// updateChargeStatus updates car status and stops charging if car disconnected
func (lp *LoadPoint) updateChargeStatus() api.ChargeStatus {
	// abort if no vehicle connected
	status, err := lp.Charger.Status()
	if err != nil {
		log.ERROR.Printf("%s charger error: %v", lp.Name, err)
		return api.StatusNone
	}
	log.TRACE.Printf("%s charger car status: %s", lp.Name, status)

	if prevStatus := lp.state.Status(); status != prevStatus {
		lp.state.SetStatus(status)

		// connected
		if prevStatus == api.StatusA {
			log.INFO.Printf("%s car connected (%s)", lp.Name, string(status))
		}

		// disconnected
		if status == api.StatusA {
			log.INFO.Printf("%s car disconnected", lp.Name)
		}

		// start charging
		if status == api.StatusC {
			lp.chargingCycle(true)
		}

		// stop charging
		if status != api.StatusC {
			lp.chargingCycle(false)
		}
	}

	return status
}

// updateChargeCurrentAndPower retrieves chargers actual current and charge meter power.
// Charger actual current is published and may update the charge power.
// This mechanism ensures that fake charge meter and charge energy are updated.
func (lp *LoadPoint) updateChargeCurrentAndPower() (current int64, power float64, err error) {
	current, err = lp.Charger.ActualCurrent()
	if err != nil {
		return 0, 0, fmt.Errorf("%s charger error: %v", lp.Name, err)
	}
	log.TRACE.Printf("%s charge current: %dA", lp.Name, current)

	power, err = lp.ChargeMeter.CurrentPower()
	if err != nil {
		return 0, 0, fmt.Errorf("%s charge meter error: %v", lp.Name, err)
	}
	log.TRACE.Printf("%s charge power: %.0fW", lp.Name, power)

	// update charge rater with power
	lp.state.SetChargePower(power)
	lp.bus.Publish(evChargePower, power)

	return current, power, nil
}

// setTargetCurrent guards setting current against changing to identical value
// and violating MaxCurrent
func (lp *LoadPoint) setTargetCurrent(targetCurrentIn int64) error {
	targetCurrent := clamp(targetCurrentIn, lp.MinCurrent, lp.MaxCurrent)
	if targetCurrent != targetCurrentIn {
		log.WARN.Printf("%s hard limit charge current: %dA", lp.Name, targetCurrent)
	}

	if lp.state.TargetCurrent() != targetCurrent {
		log.INFO.Printf("%s set charge current: %dA", lp.Name, targetCurrent)
		if err := lp.Charger.(api.ChargeController).MaxCurrent(targetCurrent); err != nil {
			return fmt.Errorf("%s charge controller error: %v", lp.Name, err)
		}

		lp.state.SetTargetCurrent(targetCurrent)
	}

	lp.bus.Publish(evChargeCurrent, targetCurrent)

	return nil
}

// Update reevaluates meters and charger state
func (lp *LoadPoint) Update(updateChan chan<- push.Event) {
	once.Do(func() { lp.prepare(updateChan) })

	// abort if dumb charge controller
	if _, chargeController := lp.Charger.(api.ChargeController); !chargeController {
		log.ERROR.Printf("%s no charge controller assigned", lp.Name)
		return
	}

	// check if car connected and ready for charging
	if lp.updateChargeStatus(); !lp.Connected() {
		return
	}

	// execute loading strategy
	var err error
	switch mode := lp.state.Mode(); mode {
	case api.ModeOff:
		err = lp.rampOff()
	case api.ModeNow:
		err = lp.rampOn(lp.MaxCurrent)
	case api.ModeMinPV, api.ModePV:
		err = lp.ApplyModePV(mode)
	}

	if err != nil {
		log.ERROR.Println(err)
	}
}

// rampUpDown moves stepwise towards target current. If target current is reached
// during this process, true is returned otherwise false.
func (lp *LoadPoint) rampUpDown(target int64) (bool, error) {
	current := lp.state.TargetCurrent()
	if current == target {
		return true, nil
	}

	var step int64
	if current < target {
		step = min(current+lp.Steepness, target)
	} else if current > target {
		step = max(current-lp.Steepness, target)
	}

	step = clamp(step, lp.MinCurrent, lp.MaxCurrent)

	if err := lp.setTargetCurrent(step); err != nil {
		return false, err
	}

	// end of ramp reached?
	if step == target {
		return true, nil
	}

	return false, nil
}

// rampOff ramps down charging current to minimum and then turns off
func (lp *LoadPoint) rampOff() error {
	if enabled := lp.updateChargerEnabled(); enabled {
		// log.DEBUG.Printf(". rampOff %d", lp.MinCurrent)
		finished, err := lp.rampUpDown(lp.MinCurrent)
		if err != nil {
			return err
		}

		if finished {
			// TODO handle chargerEnable aborted
			lp.bus.Publish(evChargeCurrent, int64(0))
			return lp.chargerEnable(false)
		}
	}

	return nil
}

// rampUp ramps up charging current to maximum and then turns off
func (lp *LoadPoint) rampOn(target int64) error {
	if enabled := lp.updateChargerEnabled(); !enabled {
		if err := lp.setTargetCurrent(lp.MinCurrent); err != nil {
			return err
		}

		return lp.chargerEnable(true)
	}

	_, err := lp.rampUpDown(target)
	return err
}

// ApplyModePV sets "minpv" or "pv" load modes
func (lp *LoadPoint) ApplyModePV(mode api.ChargeMode) error {
	// get charger current and charge meter power
	_, chargePower, err := lp.updateChargeCurrentAndPower()
	if err != nil {
		return err
	}

	var targetChargePower float64

	// use grid meter if available, pv meter else
	if lp.GridMeter != nil {
		gridPower, err := lp.GridMeter.CurrentPower()
		if err != nil {
			return fmt.Errorf("%s grid meter: %v", lp.Name, err)
		}
		log.TRACE.Printf("%s grid power: %.0fW", lp.Name, gridPower)

		// grid power must be negative for export
		deltaChargePower := -gridPower - lp.ResidualPower
		targetChargePower = chargePower + deltaChargePower

		log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW residual", lp.Name, targetChargePower, chargePower, gridPower, lp.ResidualPower)
	} else {
		pvPower, err := lp.PVMeter.CurrentPower()
		if err != nil {
			return fmt.Errorf("%s pv meter: %v", lp.Name, err)
		}
		log.TRACE.Printf("%s pv power: %.0fW", lp.Name, pvPower)

		pvPower = math.Abs(pvPower)
		targetChargePower = pvPower - lp.ResidualPower

		log.DEBUG.Printf("%s target power: %.0fW = %.0fW pv - %.0fW residual", lp.Name, targetChargePower, pvPower, lp.ResidualPower)
	}

	// get max charge current
	targetChargeCurrent := clamp(powerToCurrent(targetChargePower, lp.Voltage, lp.Phases), 0, lp.MaxCurrent)

	if targetChargeCurrent < lp.MinCurrent {
		switch mode {
		case api.ModeMinPV:
			targetChargeCurrent = lp.MinCurrent
		case api.ModePV:
			targetChargeCurrent = 0
		}
	}

	log.DEBUG.Printf("%s target current: %dA", lp.Name, targetChargeCurrent)

	if targetChargeCurrent == 0 {
		return lp.rampOff()
	}

	return lp.rampOn(targetChargeCurrent)
}
