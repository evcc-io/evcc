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
	"github.com/avast/retry-go"
	"github.com/benbjohnson/clock"
)

var (
	status   = map[bool]string{false: "disable", true: "enable"}
	presence = map[bool]string{false: "—", true: "✓"}
)

const (
	evStartCharge   = "start"   // update chargeTimer
	evStopCharge    = "stop"    // update chargeTimer
	evChargeCurrent = "current" // update fakeChargeMeter
	evChargePower   = "power"   // update chargeRater
)

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power, voltage float64, phases int64) int64 {
	return int64(power / (float64(phases) * voltage))
}

// LoadPoint is responsible for controlling charge depending on
// SoC needs and power availability.
type LoadPoint struct {
	sync.Mutex                         // guard status
	clock            clock.Clock       // mockable time
	bus              evbus.Bus         // event bus
	triggerChan      chan struct{}     // API updates
	notificationChan chan<- push.Event // notifications
	uiChan           chan<- Param      // client push messages

	Name        string
	Charger     api.Charger
	ChargeTimer api.ChargeTimer
	ChargeRater api.ChargeRater

	// meters
	GridMeter   api.Meter   // Grid usage meter
	PVMeter     api.Meter   // PV generation meter
	ChargeMeter api.Meter   // Charger usage meter
	Vehicle     api.Vehicle // Vehicle

	// options
	Steepness     int64   // Step size of current change
	Phases        int64   // Phases- required for converting power and current.
	MinCurrent    int64   // PV mode: start current	Min+PV mode: min current
	MaxCurrent    int64   // Max allowed current. Physically ensured by the charge controller
	Voltage       float64 // Operating voltage. 230V for Germany.
	ResidualPower float64 // PV meter only: household usage. Grid meter: household safety margin

	// cached state
	Mode          api.ChargeMode   // Charge mode, guarded by mutex
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
		clock:         clock.New(),
		bus:           evbus.New(),
		triggerChan:   make(chan struct{}, 1),
		Name:          "Main",
		Mode:          api.ModeOff,
		status:        api.StatusNone,
		Phases:        1,
		Voltage:       230, // V
		MinCurrent:    6,   // A
		MaxCurrent:    16,  // A
		Steepness:     10,  // A
		targetCurrent: 0,   // A
		GuardDuration: 10 * time.Minute,
	}
}

// notify sends push messages to clients
func (lp *LoadPoint) notify(event string, attributes map[string]interface{}) {
	attributes["loadpoint"] = lp.Name
	lp.notificationChan <- push.Event{
		Event:      event,
		Attributes: attributes,
	}
}

// publish sends values to UI and databases
func (lp *LoadPoint) publish(key string, val interface{}) {
	lp.uiChan <- Param{
		LoadPoint: lp.Name,
		Key:       key,
		Val:       val,
	}
}

// evChargeStartHandler sends external start event
func (lp *LoadPoint) evChargeStartHandler() {
	lp.notify(evStartCharge, map[string]interface{}{
		"mode": lp.GetMode(),
	})
}

// evChargeStartHandler sends external stop event
func (lp *LoadPoint) evChargeStopHandler() {
	energy, err := lp.ChargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s charged energy: %v", lp.Name, err)
	}

	duration, err := lp.ChargeTimer.ChargingTime()
	if err != nil {
		log.ERROR.Printf("%s charge duration: %v", lp.Name, err)
	}

	lp.notify(evStopCharge, map[string]interface{}{
		"energy":   energy,
		"duration": duration.Truncate(time.Second),
	})
}

// evChargeCurrentHandler updates proxy charge meter's charge current.
// If physical charge meter is present this handler is not used.
func (lp *LoadPoint) evChargeCurrentHandler(m *wrapper.ChargeMeter) func(para ...interface{}) {
	return func(para ...interface{}) {
		current := para[0].(int64)
		if !lp.enabled || lp.status != api.StatusC {
			current = 0
		}
		if current > 0 {
			// limit available power to generation plus consumption/ minus delivery
			availablePower := math.Abs(lp.pvPower) + lp.gridPower
			availableCurrent := int64(powerToCurrent(availablePower, lp.Voltage, lp.Phases))
			current = min(current, availableCurrent)
		}
		m.SetChargeCurrent(current)
	}
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) Prepare(uiChan chan<- Param, notificationChan chan<- push.Event) {
	lp.notificationChan = notificationChan
	lp.uiChan = uiChan

	if lp.PVMeter == nil && lp.GridMeter == nil {
		log.FATAL.Fatal("missing either PV or Grid meter - aborting")
	}

	// ensure charge meter exists
	if lp.ChargeMeter == nil {
		if mt, ok := lp.Charger.(api.Meter); ok {
			lp.ChargeMeter = mt
		} else {
			mt := &wrapper.ChargeMeter{
				Phases:  lp.Phases,
				Voltage: lp.Voltage,
			}
			_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentHandler(mt))
			_ = lp.bus.Subscribe(evStopCharge, func() {
				mt.SetChargeCurrent(0)
			})
			lp.ChargeMeter = mt
		}
	}

	// ensure charge rater exists
	if rt, ok := lp.Charger.(api.ChargeRater); ok {
		lp.ChargeRater = rt
	} else {
		rt := wrapper.NewChargeRater(lp.Name, lp.ChargeMeter)
		_ = lp.bus.Subscribe(evChargePower, rt.SetChargePower)
		_ = lp.bus.Subscribe(evStartCharge, rt.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, rt.StopCharge)
		lp.ChargeRater = rt
	}

	// ensure charge timer exists
	if ct, ok := lp.Charger.(api.ChargeTimer); ok {
		lp.ChargeTimer = ct
	} else {
		ct := wrapper.NewChargeTimer()
		_ = lp.bus.Subscribe(evStartCharge, ct.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, ct.StopCharge)
		lp.ChargeTimer = ct
	}

	// event handlers
	_ = lp.bus.Subscribe(evStartCharge, lp.evChargeStartHandler)
	_ = lp.bus.Subscribe(evStopCharge, lp.evChargeStopHandler)

	// read initial enabled state
	enabled, err := lp.Charger.Enabled()
	if err == nil {
		lp.enabled = enabled
		log.INFO.Printf("%s charger %s", lp.Name, status[lp.enabled])

		// prevent immediately disabling charger
		if lp.enabled {
			lp.guardUpdated = lp.clock.Now()
		}
	} else {
		log.ERROR.Printf("%s charger error: %v", lp.Name, err)
	}

	// set current to known value
	if err = lp.setTargetCurrent(lp.MinCurrent); err != nil {
		log.ERROR.Println(err)
	}
	lp.bus.Publish(evChargeCurrent, lp.MinCurrent)
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

	if remaining := (lp.GuardDuration - time.Since(lp.guardUpdated)).Truncate(time.Second); remaining > 0 {
		log.DEBUG.Printf("%s charger %s - contactor delay %v", lp.Name, status[enable], remaining)
		return nil
	}

	err := lp.Charger.Enable(enable)
	if err == nil {
		lp.enabled = enable // cache
		log.INFO.Printf("%s charger %s", lp.Name, status[enable])
		lp.guardUpdated = lp.clock.Now()

		// if not enabled, current will be reduced to 0 in handler
		lp.bus.Publish(evChargeCurrent, lp.MinCurrent)
	} else {
		log.DEBUG.Printf("%s charger %s", lp.Name, status[enable])
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
	log.DEBUG.Printf("%s charger status: %s", lp.Name, status)

	if prevStatus := lp.status; status != prevStatus {
		lp.status = status

		// connected
		if prevStatus == api.StatusA {
			log.INFO.Printf("%s car connected (%s)", lp.Name, string(status))
			if lp.enabled {
				// when car connected don't disable right away
				lp.guardUpdated = lp.clock.Now()
			}
		}

		// disconnected
		if status == api.StatusA {
			log.INFO.Printf("%s car disconnected", lp.Name)
		}

		lp.bus.Publish(evChargeCurrent, lp.targetCurrent)

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

// updateModePV sets "minpv" or "pv" load modes
func (lp *LoadPoint) updateModePV(mode api.ChargeMode) error {
	// grid meter will always be available, if as wrapped pv meter
	targetChargePower := lp.chargePower - lp.gridPower - lp.ResidualPower
	log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW residual", lp.Name, targetChargePower, lp.chargePower, lp.gridPower, lp.ResidualPower)

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

	log.DEBUG.Printf("%s target charge current: %dA", lp.Name, targetChargeCurrent)

	if targetChargeCurrent == 0 {
		return lp.rampOff()
	}

	return lp.rampOn(targetChargeCurrent)
}

// updateMeter updates and publishes single meter
func (lp *LoadPoint) updateMeter(name string, meter api.Meter, power *float64) error {
	value, err := meter.CurrentPower()
	if err != nil {
		log.ERROR.Printf("%s %v", lp.Name, err)
		return err
	}

	*power = value // update value if no error

	log.DEBUG.Printf("%s %s power: %.1fW", lp.Name, name, *power)
	lp.publish(name+"Power", *power)

	return nil
}

// updateMeter updates and publishes single meter
func (lp *LoadPoint) updateMeters() (err error) {
	var wg sync.WaitGroup

	var mux sync.Mutex
	retry := func(s string, m api.Meter, f *float64) {
		e := retry.Do(func() error {
			return lp.updateMeter(s, m, f)
		})
		if e != nil {
			mux.Lock()
			err = e
			mux.Unlock()
		}
		wg.Done()
	}

	wg.Add(3)
	go retry("grid", lp.GridMeter, &lp.gridPower)
	go retry("pv", lp.PVMeter, &lp.pvPower)
	go retry("charge", lp.ChargeMeter, &lp.chargePower)
	wg.Wait()

	return err
}

// update is the main control function. It reevaluates meters and charger state
func (lp *LoadPoint) update() {
	lp.updateChargeStatus()

	lp.publish("mode", string(lp.GetMode()))
	lp.publish("connected", lp.connected())
	lp.publish("charging", lp.charging)

	// catch any persistent meter update error
	meterErr := lp.updateMeters()

	// update ChargeRater here to make sure initial meter update is caught
	lp.bus.Publish(evChargeCurrent, lp.targetCurrent)
	lp.bus.Publish(evChargePower, lp.chargePower)

	// check if car connected and ready for charging
	var err error
	if !lp.connected() {
		// ensure restart at min current
		err = lp.setTargetCurrent(lp.MinCurrent)
	} else {
		// execute loading strategy
		switch mode := lp.GetMode(); mode {
		case api.ModeOff:
			err = lp.rampOff()
		case api.ModeNow:
			err = lp.rampOn(lp.MaxCurrent)
		case api.ModeMinPV, api.ModePV:
			if meterErr == nil {
				// pv modes require meter measurements
				err = lp.updateModePV(mode)
			} else {
				log.WARN.Printf("%s aborting due to meter error", lp.Name)
			}
		}
	}

	if err != nil {
		log.ERROR.Println(err)
	}

	lp.publish("chargedEnergy", 1e3*lp.chargedEnergy()) // return Wh for UI
	lp.publish("chargeDuration", lp.chargeDuration())

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
