package core

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/push"

	evbus "github.com/asaskevich/EventBus"
)

var (
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
	sync.Mutex                         // guard status
	bus              evbus.Bus         // event bus
	triggerChan      chan struct{}     // API updates
	notificationChan chan<- push.Event // notifications
	uiChan           chan<- Param      // client push messages

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

	// cached state
	Mode          api.ChargeMode   // Charge mode, garded by mux
	status        api.ChargeStatus // Charger status
	targetCurrent int64            // Allowed current. Between MinCurrent and MaxCurrent.
	enabled       bool             // Charger enabled state
	charging      bool             // Charging cycle
	gridPower     float64          // Grid power
	pvPower       float64          // PV power
	chargePower   float64          // Charging power

	// contactor switch guard
	guardUpdated  time.Time     // charger enabled/disabled timestamp
	GuardDuration time.Duration // charger enable/disable minimum holding time
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint() *LoadPoint {
	return &LoadPoint{
		bus:           evbus.New(),
		triggerChan:   make(chan struct{}, 1),
		Name:          "Main",
		Phases:        1,
		Voltage:       230, // V
		MinCurrent:    6,   // A
		MaxCurrent:    16,  // A
		Steepness:     1,   // A
		targetCurrent: 0,   // A
		GuardDuration: 10 * time.Minute,
	}
}

// connected returns the EVs connection state
func (lp *LoadPoint) connected() bool {
	return lp.status == api.StatusB || lp.status == api.StatusC
}

// chargerEnable switches charging on or off. Minimum cycle duration is guaranteed.
func (lp *LoadPoint) chargerEnable(enable bool) error {
	if lp.targetCurrent != 0 && lp.targetCurrent != lp.MinCurrent {
		log.FATAL.Fatal("charger enable/disable called without setting min current first")
	}

	if time.Since(lp.guardUpdated) < lp.GuardDuration {
		log.TRACE.Printf("%s min cycle time violated - ignoring charger %s", lp.Name, status[enable])
		return nil
	}

	err := lp.Charger.Enable(enable)
	if err == nil {
		lp.enabled = enable // cache
		log.INFO.Printf("%s charger %s", lp.Name, status[enable])
		lp.guardUpdated = time.Now()

		if !enable {
			lp.bus.Publish(evChargeCurrent, int64(0))
		}
	}
	return err
}

// chargingCycle detects charge cycle start and stop events and manages the
// charge energy counter and charge timer. It guards against duplicate invocation.
func (lp *LoadPoint) chargingCycle(enable bool) {
	if enable == lp.charging {
		return
	}

	lp.charging = enable

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
	log.TRACE.Printf("%s charger status: %s", lp.Name, status)

	if prevStatus := lp.status; status != prevStatus {
		lp.status = status

		// connected
		if prevStatus == api.StatusA {
			log.INFO.Printf("%s car connected (%s)", lp.Name, string(status))
		}

		// disconnected
		if status == api.StatusA {
			log.INFO.Printf("%s car disconnected", lp.Name)
		}

		// start/stop charging cycle
		lp.chargingCycle(status == api.StatusC)
	}

	return status
}

// setTargetCurrent guards setting current against changing to identical value
// and violating MaxCurrent
func (lp *LoadPoint) setTargetCurrent(targetCurrentIn int64) error {
	targetCurrent := clamp(targetCurrentIn, lp.MinCurrent, lp.MaxCurrent)
	if targetCurrent != targetCurrentIn {
		log.WARN.Printf("%s hard limit charge current: %dA", lp.Name, targetCurrent)
	}

	if lp.targetCurrent != targetCurrent {
		log.DEBUG.Printf("%s set charge current: %dA", lp.Name, targetCurrent)
		if err := lp.Charger.MaxCurrent(targetCurrent); err != nil {
			return fmt.Errorf("%s charge controller error: %v", lp.Name, err)
		}

		lp.targetCurrent = targetCurrent // cache
	}

	lp.bus.Publish(evChargeCurrent, targetCurrent)

	return nil
}

// rampUpDown moves stepwise towards target current. If target current is reached
// during this process, true is returned otherwise false.
func (lp *LoadPoint) rampUpDown(target int64) (bool, error) {
	current := lp.targetCurrent
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
	if lp.enabled {
		finished, err := lp.rampUpDown(lp.MinCurrent)
		if err != nil {
			return err
		}

		if finished {
			return lp.chargerEnable(false)
		}
	}

	return nil
}

// rampUp ramps up charging current to maximum and then turns off
func (lp *LoadPoint) rampOn(target int64) error {
	if !lp.enabled {
		if err := lp.setTargetCurrent(lp.MinCurrent); err != nil {
			return err
		}

		return lp.chargerEnable(true)
	}

	_, err := lp.rampUpDown(target)
	return err
}

func (lp *LoadPoint) targetChargePower() float64 {
	var targetChargePower float64

	// use grid meter if available, pv meter else
	if lp.GridMeter != nil {
		// grid power must be negative for export
		targetChargePower = lp.chargePower - lp.gridPower - lp.ResidualPower
		log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW residual", lp.Name, targetChargePower, lp.chargePower, lp.gridPower, lp.ResidualPower)
	} else {
		targetChargePower = math.Abs(lp.pvPower) - lp.ResidualPower
		log.DEBUG.Printf("%s target power: %.0fW = %.0fW pv - %.0fW residual", lp.Name, targetChargePower, lp.pvPower, lp.ResidualPower)
	}

	return targetChargePower
}

// updateModePV sets "minpv" or "pv" load modes
func (lp *LoadPoint) updateModePV(mode api.ChargeMode) error {
	targetChargePower := lp.targetChargePower()

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

	log.TRACE.Printf("%s target charge current: %dA", lp.Name, targetChargeCurrent)

	if targetChargeCurrent == 0 {
		return lp.rampOff()
	}

	return lp.rampOn(targetChargeCurrent)
}

// updateMeter updates and publishes single meter
func (lp *LoadPoint) updateMeter(name string, meter api.Meter, power *float64) {
	var err error
	*power, err = meter.CurrentPower()
	if err != nil {
		log.ERROR.Printf("%s %v", lp.Name, err)
		return
	}

	log.TRACE.Printf("%s %s power: %.1fW", lp.Name, name, *power)
	lp.uiChan <- Param{Key: name + "Power", Val: *power}
}

// update is the main control function. It reevaluates meters and charger state
func (lp *LoadPoint) update() {
	lp.updateChargeStatus()

	lp.uiChan <- Param{Key: "mode", Val: string(lp.GetMode())}
	lp.uiChan <- Param{Key: "connected", Val: lp.connected()}
	lp.uiChan <- Param{Key: "charging", Val: lp.charging}

	lp.updateMeter("grid", lp.GridMeter, &lp.gridPower)
	lp.updateMeter("pv", lp.PVMeter, &lp.pvPower)
	lp.updateMeter("charge", lp.ChargeMeter, &lp.chargePower)

	// update ChargeRater
	lp.bus.Publish(evChargePower, lp.chargePower)

	// check if car connected and ready for charging
	if !lp.connected() {
		return
	}

	// execute loading strategy
	var err error
	switch mode := lp.GetMode(); mode {
	case api.ModeOff:
		err = lp.rampOff()
	case api.ModeNow:
		err = lp.rampOn(lp.MaxCurrent)
	case api.ModeMinPV, api.ModePV:
		if lp.charging {
			err = lp.updateModePV(mode)
		}
	}

	if err != nil {
		log.ERROR.Println(err)
	}

	lp.uiChan <- Param{Key: "chargedEnergy", Val: 1e3 * lp.chargedEnergy()}
	lp.uiChan <- Param{Key: "chargeDuration", Val: lp.chargeDuration()}

	lp.publishSoC()
}

// Run is the loadpoint main control loop. It reacts to trigger events by
// updating measurements and executing control logic.
func (lp *LoadPoint) Run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	lp.triggerChan <- struct{}{} // start immediately

	for {
		select {
		case <-ticker.C:
			lp.update()
		case <-lp.triggerChan:
			lp.update()
			ticker.Stop()
			ticker = time.NewTicker(interval)
		}
	}
}
