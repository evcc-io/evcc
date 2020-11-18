package core

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"

	evbus "github.com/asaskevich/EventBus"
	"github.com/avast/retry-go"
	"github.com/benbjohnson/clock"
)

const (
	evChargeStart       = "start"      // update chargeTimer
	evChargeStop        = "stop"       // update chargeTimer
	evChargeCurrent     = "current"    // update fakeChargeMeter
	evChargePower       = "power"      // update chargeRater
	evVehicleConnect    = "connect"    // vehicle connected
	evVehicleDisconnect = "disconnect" // vehicle disconnected

	minActiveCurrent = 1.0 // minimum current at which a phase is treated as active
)

// SoCConfig defines soc settings, estimation and update behaviour
type SoCConfig struct {
	AlwaysUpdate bool  `mapstructure:"alwaysUpdate"`
	Levels       []int `mapstructure:"levels"`
	Estimate     bool  `mapstructure:"estimate"`
	Min          int   `mapstructure:"min"`    // Default minimum SoC, guarded by mutex
	Target       int   `mapstructure:"target"` // Default target SoC, guarded by mutex
}

// ThresholdConfig defines enable/disable hysteresis parameters
type ThresholdConfig struct {
	Delay     time.Duration
	Threshold float64
}

// LoadPoint is responsible for controlling charge depending on
// SoC needs and power availability.
type LoadPoint struct {
	clock    clock.Clock       // mockable time
	bus      evbus.Bus         // event bus
	pushChan chan<- push.Event // notifications
	uiChan   chan<- util.Param // client push messages
	lpChan   chan<- *LoadPoint // update requests
	log      *util.Logger

	// exposed public configuration
	sync.Mutex                // guard status
	Mode       api.ChargeMode `mapstructure:"mode"` // Charge mode, guarded by mutex

	Title       string   `mapstructure:"title"`    // UI title
	Phases      int64    `mapstructure:"phases"`   // Phases- required for converting power and current
	ChargerRef  string   `mapstructure:"charger"`  // Charger reference
	VehicleRef  string   `mapstructure:"vehicle"`  // Vehicle reference
	VehiclesRef []string `mapstructure:"vehicles"` // Vehicles reference
	Meters      struct {
		ChargeMeterRef string `mapstructure:"charge"` // Charge meter reference
	}
	SoC          SoCConfig
	OnDisconnect struct {
		Mode      api.ChargeMode `mapstructure:"mode"`      // Charge mode to apply when car disconnected
		TargetSoC int            `mapstructure:"targetSoC"` // Target SoC to apply when car disconnected
	}
	Enable, Disable ThresholdConfig

	handler       Handler
	HandlerConfig `mapstructure:",squash"` // handle charger state and current

	chargeTimer api.ChargeTimer
	chargeRater api.ChargeRater

	chargeMeter  api.Meter     // Charger usage meter
	vehicle      api.Vehicle   // Currently active vehicle
	vehicles     []api.Vehicle // Assigned vehicles
	socEstimator *wrapper.SocEstimator

	// cached state
	status        api.ChargeStatus // Charger status
	charging      bool             // Charging cycle
	chargePower   float64          // Charging power
	connectedTime time.Time        // Time when vehicle was connected
	pvTimer       time.Time        // PV enabled/disable timer

	socCharge      float64       // Vehicle SoC
	chargedEnergy  float64       // Charged energy while connected in Wh
	chargeDuration time.Duration // Charge duration
}

// NewLoadPointFromConfig creates a new loadpoint
func NewLoadPointFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) (*LoadPoint, error) {
	lp := NewLoadPoint(log)
	if err := util.DecodeOther(other, &lp); err != nil {
		return nil, err
	}

	// set sane defaults
	lp.Mode = api.ChargeModeString(string(lp.Mode))
	lp.OnDisconnect.Mode = api.ChargeModeString(string(lp.OnDisconnect.Mode))

	sort.Ints(lp.SoC.Levels)
	if lp.SoC.Target == 0 {
		lp.SoC.Target = lp.OnDisconnect.TargetSoC // use disconnect value as default soc
		if lp.SoC.Target == 0 {
			lp.SoC.Target = 100
		}

		if len(lp.SoC.Levels) > 0 {
			lp.SoC.Target = lp.SoC.Levels[len(lp.SoC.Levels)-1]
		}
	}

	if lp.Meters.ChargeMeterRef != "" {
		lp.chargeMeter = cp.Meter(lp.Meters.ChargeMeterRef)
	}

	// multiple vehicles
	for _, ref := range lp.VehiclesRef {
		vehicle := cp.Vehicle(ref)
		lp.vehicles = append(lp.vehicles, vehicle)
	}

	// single vehicle
	if lp.VehicleRef != "" {
		vehicle := cp.Vehicle(lp.VehicleRef)
		lp.vehicles = append(lp.vehicles, vehicle)
	}

	if lp.ChargerRef == "" {
		return nil, errors.New("missing charger")
	}
	charger := cp.Charger(lp.ChargerRef)
	lp.configureChargerType(charger)

	if lp.Enable.Threshold > lp.Disable.Threshold {
		log.WARN.Printf("PV mode enable threshold (%.0fW) is larger than disable threshold (%.0fW)", lp.Enable.Threshold, lp.Disable.Threshold)
	}

	lp.handler = &ChargerHandler{
		log:           lp.log,
		clock:         lp.clock,
		bus:           lp.bus,
		charger:       charger,
		HandlerConfig: lp.HandlerConfig,
	}

	return lp, nil
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint(log *util.Logger) *LoadPoint {
	clock := clock.New()
	bus := evbus.New()

	lp := &LoadPoint{
		log:    log,   // logger
		clock:  clock, // mockable time
		bus:    bus,   // event bus
		Mode:   api.ModeOff,
		Phases: 1,
		status: api.StatusNone,
		HandlerConfig: HandlerConfig{
			MinCurrent:    6,  // A
			MaxCurrent:    16, // A
			Sensitivity:   10, // A
			GuardDuration: 5 * time.Minute,
		},
	}

	return lp
}

// requestUpdate requests site to update this loadpoint
func (lp *LoadPoint) requestUpdate() {
	select {
	case lp.lpChan <- lp: // request loadpoint update
	default:
	}
}

// configureChargerType ensures that chargeMeter, Rate and Timer can use charger capabilities
func (lp *LoadPoint) configureChargerType(charger api.Charger) {
	// ensure charge meter exists
	if lp.chargeMeter == nil {
		if mt, ok := charger.(api.Meter); ok {
			lp.chargeMeter = mt
		} else {
			mt := &wrapper.ChargeMeter{}
			_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentWrappedMeterHandler)
			_ = lp.bus.Subscribe(evChargeStop, func() { mt.SetPower(0) })
			lp.chargeMeter = mt
		}
	}

	// ensure charge rater exists
	if rt, ok := charger.(api.ChargeRater); ok {
		lp.chargeRater = rt
	} else {
		rt := wrapper.NewChargeRater(lp.log, lp.chargeMeter)
		_ = lp.bus.Subscribe(evChargePower, rt.SetChargePower)
		_ = lp.bus.Subscribe(evVehicleConnect, func() { rt.StartCharge(false) })
		_ = lp.bus.Subscribe(evChargeStart, func() { rt.StartCharge(true) })
		_ = lp.bus.Subscribe(evChargeStop, rt.StopCharge)
		lp.chargeRater = rt
	}

	// ensure charge timer exists
	if ct, ok := charger.(api.ChargeTimer); ok {
		lp.chargeTimer = ct
	} else {
		ct := wrapper.NewChargeTimer()
		_ = lp.bus.Subscribe(evVehicleConnect, func() { ct.StartCharge(false) })
		_ = lp.bus.Subscribe(evChargeStart, func() { ct.StartCharge(true) })
		_ = lp.bus.Subscribe(evChargeStop, ct.StopCharge)
		lp.chargeTimer = ct
	}
}

// notify sends push messages to clients
func (lp *LoadPoint) notify(event string) {
	lp.pushChan <- push.Event{Event: event}
}

// publish sends values to UI and databases
func (lp *LoadPoint) publish(key string, val interface{}) {
	if lp.uiChan != nil {
		lp.uiChan <- util.Param{Key: key, Val: val}
	}
}

// evChargeStartHandler sends external start event
func (lp *LoadPoint) evChargeStartHandler() {
	lp.log.INFO.Println("start charging ->")
	lp.notify(evChargeStart)
}

// evChargeStopHandler sends external stop event
func (lp *LoadPoint) evChargeStopHandler() {
	lp.log.INFO.Println("stop charging <-")
	lp.notify(evChargeStop)
}

// evVehicleConnectHandler sends external start event
func (lp *LoadPoint) evVehicleConnectHandler() {
	lp.log.INFO.Printf("car connected")

	// energy
	lp.chargedEnergy = 0
	lp.publish("chargedEnergy", lp.chargedEnergy)

	// duration
	lp.connectedTime = lp.clock.Now()
	lp.publish("connectedDuration", 0)

	// soc estimation reset on car change
	if lp.socEstimator != nil {
		lp.socEstimator.Reset()
	}

	lp.notify(evVehicleConnect)
}

// evVehicleDisconnectHandler sends external start event
func (lp *LoadPoint) evVehicleDisconnectHandler() {
	lp.log.INFO.Println("car disconnected")

	// energy and duration
	lp.publish("chargedEnergy", lp.chargedEnergy)
	lp.publish("connectedDuration", lp.clock.Since(lp.connectedTime))

	lp.notify(evVehicleDisconnect)

	// set default mode on disconnect
	if lp.OnDisconnect.Mode != "" && lp.GetMode() != api.ModeOff {
		lp.SetMode(lp.OnDisconnect.Mode)
	}
	if lp.OnDisconnect.TargetSoC != 0 {
		_ = lp.SetTargetSoC(lp.OnDisconnect.TargetSoC)
	}
}

// evChargeCurrentHandler updates the dummy charge meter's charge power. This simplifies the main flow
func (lp *LoadPoint) evChargeCurrentHandler(current int64) {
	lp.publish("chargeCurrent", current)
}

// evChargeCurrentWrappedMeterHandler updates the dummy charge meter's charge power.
// This simplifies the main flow where the charge meter can always be treated as present.
// It assumes that the charge meter cannot consume more than total household consumption.
// If physical charge meter is present this handler is not used.
// The actual value is published by the evChargeCurrentHandler
func (lp *LoadPoint) evChargeCurrentWrappedMeterHandler(current int64) {
	power := float64(current*lp.Phases) * Voltage

	if !lp.handler.Enabled() || lp.status != api.StatusC {
		// if disabled we cannot be charging
		power = 0
	}
	// TODO
	// else if power > 0 && lp.Site.pvMeter != nil {
	// 	// limit charge power to generation plus grid consumption/ minus grid delivery
	// 	// as the charger cannot have consumed more than that
	// 	// consumedPower := consumedPower(lp.pvPower, lp.batteryPower, lp.gridPower)
	// 	consumedPower := lp.Site.consumedPower()
	// 	power = math.Min(power, consumedPower)
	// }

	// handler only called if charge meter was replaced by dummy
	lp.chargeMeter.(*wrapper.ChargeMeter).SetPower(power)
}

// Name returns the human-readable loadpoint title
func (lp *LoadPoint) Name() string {
	return lp.Title
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event, lpChan chan<- *LoadPoint) {
	lp.uiChan = uiChan
	lp.pushChan = pushChan
	lp.lpChan = lpChan

	// event handlers
	_ = lp.bus.Subscribe(evChargeStart, lp.evChargeStartHandler)
	_ = lp.bus.Subscribe(evChargeStop, lp.evChargeStopHandler)
	_ = lp.bus.Subscribe(evVehicleConnect, lp.evVehicleConnectHandler)
	_ = lp.bus.Subscribe(evVehicleDisconnect, lp.evVehicleDisconnectHandler)
	_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentHandler)

	// publish initial values
	lp.publish("title", lp.Title)
	lp.publish("minCurrent", lp.MinCurrent)
	lp.publish("maxCurrent", lp.MaxCurrent)
	lp.publish("phases", lp.Phases)
	lp.publish("activePhases", lp.Phases)
	lp.publish("soc", len(lp.vehicles) > 0)

	lp.Lock()
	lp.publish("mode", lp.Mode)
	lp.publish("targetSoC", lp.SoC.Target)
	lp.publish("minSoC", lp.SoC.Min)
	lp.publish("socLevels", lp.SoC.Levels)
	lp.Unlock()

	// use first vehicle for estimator
	// run during prepare() to ensure cache has been attached
	if len(lp.vehicles) > 0 {
		lp.setActiveVehicle(lp.vehicles[0])
	}

	// prepare charger status
	lp.handler.Prepare()
}

// connected returns the EVs connection state
func (lp *LoadPoint) connected() bool {
	return lp.status == api.StatusB || lp.status == api.StatusC
}

// targetSocReached checks if target is configured and reached.
// If vehicle is not configured this will always return false
func (lp *LoadPoint) targetSocReached() bool {
	return lp.vehicle != nil &&
		lp.SoC.Target > 0 &&
		lp.SoC.Target < 100 &&
		lp.socCharge >= float64(lp.SoC.Target)
}

// minSocNotReached checks if minimum is configured and not reached.
// If vehicle is not configured this will always return true
func (lp *LoadPoint) minSocNotReached() bool {
	return lp.vehicle != nil &&
		lp.SoC.Min > 0 &&
		lp.socCharge < float64(lp.SoC.Min)
}

// climateActive checks if vehicle has active climate request
func (lp *LoadPoint) climateActive() bool {
	if cl, ok := lp.vehicle.(api.Climater); ok {
		active, outsideTemp, targetTemp, err := cl.Climater()
		if err == nil {
			lp.log.DEBUG.Printf("climater active: %v, target temp: %.1f°C, outside temp: %.1f°C", active, targetTemp, outsideTemp)

			status := "off"
			if active {
				status = "on"

				switch {
				case outsideTemp < targetTemp:
					status = "heating"
				case outsideTemp > targetTemp:
					status = "cooling"
				}
			}

			lp.publish("climater", status)
			return active
		}

		lp.log.ERROR.Printf("climater: %v", err)
	}

	return false
}

// setActiveVehicle assigns currently active vehicle and configures soc estimator
func (lp *LoadPoint) setActiveVehicle(vehicle api.Vehicle) {
	if lp.vehicle != nil {
		lp.log.INFO.Printf("vehicle updated: %s -> %s", lp.vehicle.Title(), vehicle.Title())
	}

	lp.vehicle = vehicle
	lp.socEstimator = wrapper.NewSocEstimator(lp.log, vehicle, lp.SoC.Estimate)

	lp.publish("socTitle", lp.vehicle.Title())
	lp.publish("socCapacity", lp.vehicle.Capacity())
}

// findActiveVehicle validates if the active vehicle is still connected to the loadpoint
func (lp *LoadPoint) findActiveVehicle() {
	if len(lp.vehicles) <= 1 {
		return
	}

	if vs, ok := lp.vehicle.(api.VehicleStatus); ok {
		status, err := vs.Status()

		if err == nil {
			lp.log.DEBUG.Printf("vehicle status: %s (%s)", status, lp.vehicle.Title())

			// vehicle is plugged or charging, so it should be the right one
			if status == api.StatusB || status == api.StatusC {
				return
			}

			for _, vehicle := range lp.vehicles {
				if vehicle == lp.vehicle {
					continue
				}

				if vs, ok := vehicle.(api.VehicleStatus); ok {
					status, err := vs.Status()

					if err == nil {
						lp.log.DEBUG.Printf("vehicle status: %s (%s)", status, vehicle.Title())

						// vehicle is plugged or charging, so it should be the right one
						if status == api.StatusB || status == api.StatusC {
							lp.setActiveVehicle(vehicle)
							return
						}
					}
				}
			}
		}
	}
}

// updateChargerStatus updates charger status and detects car connected/disconnected events
func (lp *LoadPoint) updateChargerStatus() error {
	status, err := lp.handler.Status()
	if err != nil {
		return err
	}

	lp.log.DEBUG.Printf("charger status: %s", status)

	if prevStatus := lp.status; status != prevStatus {
		lp.status = status

		// changed from A - connected
		if prevStatus == api.StatusA {
			lp.bus.Publish(evVehicleConnect)
		}

		// changed to C - start/stop charging cycle - handle before disconnect to update energy
		if lp.charging = status == api.StatusC; lp.charging {
			lp.bus.Publish(evChargeStart)
		} else if prevStatus == api.StatusC {
			lp.bus.Publish(evChargeStop)
		}

		// changed to A - disconnected
		if status == api.StatusA {
			lp.bus.Publish(evVehicleDisconnect)
		}

		// update whenever there is a state change
		lp.bus.Publish(evChargeCurrent, lp.handler.TargetCurrent())
	}

	return nil
}

// detectPhases uses MeterCurrent interface to count phases with current >=1A
func (lp *LoadPoint) detectPhases() {
	phaseMeter, ok := lp.chargeMeter.(api.MeterCurrent)
	if !ok {
		return
	}

	i1, i2, i3, err := phaseMeter.Currents()
	if err != nil {
		lp.log.ERROR.Printf("charge meter error: %v", err)
		return
	}

	currents := []float64{i1, i2, i3}
	lp.log.TRACE.Printf("charge currents: %.3gA", currents)
	lp.publish("chargeCurrents", currents)

	if lp.charging {
		var phases int64
		for _, i := range currents {
			if i >= minActiveCurrent {
				phases++
			}
		}

		if phases > 0 {
			lp.Phases = phases
			lp.log.DEBUG.Printf("detected phases: %dp %.3gA", lp.Phases, currents)

			lp.publish("activePhases", lp.Phases)
		}
	}
}

// pvDisableTimer puts the pv enable/disable timer into elapsed state
func (lp *LoadPoint) pvDisableTimer() {
	lp.pvTimer = time.Now().Add(-lp.Disable.Delay)
}

// pvMaxCurrent calculates the maximum target current for PV mode
func (lp *LoadPoint) pvMaxCurrent(mode api.ChargeMode, sitePower float64) int64 {
	// calculate target charge current from delta power and actual current
	effectiveCurrent := lp.handler.TargetCurrent()
	if lp.status != api.StatusC {
		effectiveCurrent = 0
	}
	deltaCurrent := powerToCurrent(-sitePower, lp.Phases)
	targetCurrent := clamp(effectiveCurrent+deltaCurrent, 0, lp.MaxCurrent)

	lp.log.DEBUG.Printf("max charge current: %dA = %dA + %dA (%.0fW @ %dp)", targetCurrent, effectiveCurrent, deltaCurrent, sitePower, lp.Phases)

	// in MinPV mode return at least minCurrent
	if mode == api.ModeMinPV && targetCurrent < lp.MinCurrent {
		return lp.MinCurrent
	}

	// in PV mode disable charger if car not charging and minCurrent not possible
	if mode == api.ModePV && lp.status != api.StatusC {
		lp.pvTimer = time.Time{}

		if targetCurrent < lp.MinCurrent {
			return 0
		}

		return lp.MinCurrent
	}

	// read only once to simplify testing
	enabled := lp.handler.Enabled()

	if mode == api.ModePV && enabled && targetCurrent < lp.MinCurrent {
		// kick off disable sequence
		if sitePower >= lp.Disable.Threshold {
			lp.log.DEBUG.Printf("site power %.0fW >= disable threshold %.0fW", sitePower, lp.Disable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("start pv disable timer: %v", lp.Disable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Disable.Delay {
				lp.log.DEBUG.Println("pv disable timer elapsed")
				return 0
			}

			lp.log.DEBUG.Printf("pv disable timer remaining: %v", (lp.Disable.Delay - elapsed).Round(time.Second))
		} else {
			// reset timer
			lp.pvTimer = lp.clock.Now()
		}

		return lp.MinCurrent
	}

	if mode == api.ModePV && !enabled {
		// kick off enable sequence
		if targetCurrent >= lp.MinCurrent ||
			(lp.Enable.Threshold != 0 && sitePower <= lp.Enable.Threshold) {
			lp.log.DEBUG.Printf("site power %.0fW < enable threshold %.0fW", sitePower, lp.Enable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("start pv enable timer: %v", lp.Enable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Enable.Delay {
				lp.log.DEBUG.Println("pv enable timer elapsed")
				return lp.MinCurrent
			}

			lp.log.DEBUG.Printf("pv enable timer remaining: %v", (lp.Enable.Delay - elapsed).Round(time.Second))
		} else {
			// reset timer
			lp.pvTimer = lp.clock.Now()
		}

		return 0
	}

	// reset timer to disabled state
	lp.log.DEBUG.Printf("pv timer reset")
	lp.pvTimer = time.Time{}

	return targetCurrent
}

// updateChargeMete updates and publishes single meter
func (lp *LoadPoint) updateChargeMeter() {
	err := retry.Do(func() error {
		value, err := lp.chargeMeter.CurrentPower()
		if err != nil {
			return err
		}

		lp.chargePower = value // update value if no error
		lp.log.DEBUG.Printf("charge power: %.0fW", value)
		lp.publish("chargePower", value)

		return nil
	}, retryOptions...)

	if err != nil {
		err = fmt.Errorf("updating charge meter: %v", err)
		lp.log.ERROR.Printf("%v", err)
	}
}

// publish charged energy and duration
func (lp *LoadPoint) publishChargeProgress() {
	if f, err := lp.chargeRater.ChargedEnergy(); err == nil {
		lp.chargedEnergy = 1e3 * f // convert to Wh
	} else {
		lp.log.ERROR.Printf("charge rater error: %v", err)
	}

	if d, err := lp.chargeTimer.ChargingTime(); err == nil {
		lp.chargeDuration = d.Round(time.Second)
	} else {
		lp.log.ERROR.Printf("charge timer error: %v", err)
	}

	lp.publish("chargedEnergy", lp.chargedEnergy)
	lp.publish("chargeDuration", lp.chargeDuration)
}

// publish state of charge and remaining charge duration
func (lp *LoadPoint) publishSoC() {
	if lp.socEstimator == nil {
		return
	}

	if lp.SoC.AlwaysUpdate || lp.connected() {
		f, err := lp.socEstimator.SoC(lp.chargedEnergy)
		if err == nil {
			lp.socCharge = math.Trunc(f)
			lp.log.DEBUG.Printf("vehicle soc: %.0f%%", lp.socCharge)
			lp.publish("socCharge", lp.socCharge)

			chargeEstimate := time.Duration(-1)
			if lp.charging {
				chargeEstimate = lp.socEstimator.RemainingChargeDuration(lp.chargePower, lp.SoC.Target)
			}
			lp.publish("chargeEstimate", chargeEstimate)

			chargeRemainingEnergy := 1e3 * lp.socEstimator.RemainingChargeEnergy(lp.SoC.Target)
			lp.publish("chargeRemainingEnergy", chargeRemainingEnergy)

			return
		}

		lp.log.ERROR.Printf("vehicle error: %v", err)
	}

	lp.publish("socCharge", -1)
	lp.publish("chargeEstimate", time.Duration(-1))
}

// Update is the main control function. It reevaluates meters and charger state
func (lp *LoadPoint) Update(sitePower float64) {
	mode := lp.GetMode()
	lp.publish("mode", string(mode))

	// read and publish meters first
	lp.updateChargeMeter()

	// update ChargeRater here to make sure initial meter update is caught
	lp.bus.Publish(evChargeCurrent, lp.handler.TargetCurrent())
	lp.bus.Publish(evChargePower, lp.chargePower)

	// update progress and soc before status is updated
	lp.publishChargeProgress()

	// read and publish status
	if err := lp.updateChargerStatus(); err != nil {
		lp.log.ERROR.Printf("charge controller error: %v", err)
		return
	}

	lp.publish("connected", lp.connected())
	lp.publish("charging", lp.charging)

	// update active vehicle and publish soc
	// must be run after updating charger status to make sure
	// initial update of connected state matches charger status
	lp.findActiveVehicle()
	lp.publishSoC()

	// sync settings with charger
	if lp.status != api.StatusA {
		lp.handler.SyncEnabled()
	}

	// phase detection
	lp.detectPhases()

	// check if car connected and ready for charging
	var err error

	// execute loading strategy
	switch {
	case !lp.connected():
		// always disable charger if not connected
		// https://github.com/andig/evcc/issues/105
		err = lp.handler.Ramp(0)

	case lp.targetSocReached():
		var targetCurrent int64 // zero disables
		if lp.climateActive() {
			targetCurrent = lp.MinCurrent
		}
		err = lp.handler.Ramp(targetCurrent, true)

	case mode == api.ModeOff:
		err = lp.handler.Ramp(0, true)

	case lp.minSocNotReached():
		err = lp.handler.Ramp(lp.MaxCurrent, true)
		lp.pvDisableTimer() // let PV mode disable immediately afterwards

	case mode == api.ModeNow:
		err = lp.handler.Ramp(lp.MaxCurrent, true)

	case mode == api.ModeMinPV || mode == api.ModePV:
		targetCurrent := lp.pvMaxCurrent(mode, sitePower)
		lp.log.DEBUG.Printf("target charge current: %dA", targetCurrent)

		var required bool // false
		if targetCurrent == 0 && lp.climateActive() {
			targetCurrent = lp.MinCurrent
			required = true
		}

		err = lp.handler.Ramp(targetCurrent, required)
	}

	if err != nil {
		lp.log.ERROR.Println(err)
	}
}
