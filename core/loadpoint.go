package core

import (
	"math"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	"github.com/pkg/errors"

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

// Config contains the public loadpoint configuration
type Config struct {
	Mode api.ChargeMode // Charge mode, guarded by mutex

	// options
	Phases        int64   // Phases- required for converting power and current.
	Voltage       float64 // Operating voltage. 230V for Germany.
	ResidualPower float64 // PV meter only: household usage. Grid meter: household safety margin

	ChargerRef string `mapstructure:"charger"` // Charger reference
	VehicleRef string `mapstructure:"vehicle"` // Vehicle reference

	Meters MetersConfig // Meter references
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef    string `mapstructure:"grid"`    // Grid usage meter reference
	ChargeMeterRef  string `mapstructure:"charge"`  // Charger usage meter reference
	PVMeterRef      string `mapstructure:"pv"`      // PV generation meter reference
	BatteryMeterRef string `mapstructure:"battery"` // Battery charging meter reference
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

	Config         `mapstructure:",squash"` // exposed public configuration
	ChargerHandler `mapstructure:",squash"` // handle charger state and current

	chargeTimer api.ChargeTimer
	chargeRater api.ChargeRater

	// meters
	gridMeter    api.Meter   // Grid usage meter
	pvMeter      api.Meter   // PV generation meter
	batteryMeter api.Meter   // Battery charging meter
	chargeMeter  api.Meter   // Charger usage meter
	vehicle      api.Vehicle // Vehicle

	// cached state
	status       api.ChargeStatus // Charger status
	charging     bool             // Charging cycle
	gridPower    float64          // Grid power
	pvPower      float64          // PV power
	batteryPower float64          // Battery charge power
	chargePower  float64          // Charging power
}

// configProvider gives access to configuration repository
type configProvider interface {
	Meter(string) api.Meter
	Charger(string) api.Charger
	Vehicle(string) api.Vehicle
}

// NewLoadPointFromConfig creates a new loadpoint
func NewLoadPointFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) *LoadPoint {
	lp := NewLoadPoint()
	util.DecodeOther(log, other, &lp)

	if lp.ChargerRef != "" {
		lp.charger = cp.Charger(lp.ChargerRef)
	} else {
		log.FATAL.Fatal("config: missing charger")
	}
	if lp.Meters.PVMeterRef == "" && lp.Meters.GridMeterRef == "" {
		log.FATAL.Fatal("config: missing either pv or grid meter")
	}
	if lp.Meters.GridMeterRef != "" {
		lp.gridMeter = cp.Meter(lp.Meters.GridMeterRef)
	}
	if lp.Meters.ChargeMeterRef != "" {
		lp.chargeMeter = cp.Meter(lp.Meters.ChargeMeterRef)
	}
	if lp.Meters.PVMeterRef != "" {
		lp.pvMeter = cp.Meter(lp.Meters.PVMeterRef)
	}
	if lp.Meters.BatteryMeterRef != "" {
		lp.batteryMeter = cp.Meter(lp.Meters.BatteryMeterRef)
	}
	if lp.VehicleRef != "" {
		lp.vehicle = cp.Vehicle(lp.VehicleRef)
	}

	return lp
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint() *LoadPoint {
	clock := clock.New()
	bus := evbus.New()

	lp := &LoadPoint{
		clock:       clock, // mockable time
		bus:         bus,   // event bus
		triggerChan: make(chan struct{}, 1),
		Config: Config{
			Mode:    api.ModeOff,
			Phases:  1,
			Voltage: 230, // V
		},
		status:         api.StatusNone,
		ChargerHandler: NewChargerHandler("main", clock, bus),
	}

	return lp
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
	energy, err := lp.chargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s charged energy: %v", lp.Name, err)
	}

	duration, err := lp.chargeTimer.ChargingTime()
	if err != nil {
		log.ERROR.Printf("%s charge duration: %v", lp.Name, err)
	}

	lp.notify(evStopCharge, map[string]interface{}{
		"energy":   energy,
		"duration": duration.Truncate(time.Second),
	})
}

// consumedPower estimates how much power the charger might have consumed given it was the only load
// negative values mean pv: production, battery: charging, grid: export
func consumedPower(pv, battery, grid float64) float64 {
	return math.Abs(pv) + battery + grid
}

// evChargeCurrentHandler updates the dummy charge meter's charge power. This simplifies the main flow
// where the charge meter can always be treated as present.  It assumes that the charge meter cannot consume
// more than total household consumption. If physical charge meter is present this handler is not used.
func (lp *LoadPoint) evChargeCurrentHandler(current int64) {
	power := float64(current*lp.Phases) * lp.Voltage

	if !lp.enabled || lp.status != api.StatusC {
		// if disabled we cannot be charging
		power = 0
	} else if power > 0 && lp.pvMeter != nil {
		// limit charge power to generation plus grid consumption/ minus grid delivery
		// as the charger cannot have consumed more than that
		consumedPower := consumedPower(lp.pvPower, lp.batteryPower, lp.gridPower)
		power = math.Min(power, consumedPower)
	}

	// handler only called if charge meter was replaced by dummy
	lp.chargeMeter.(*wrapper.ChargeMeter).SetPower(power)
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) Prepare(uiChan chan<- Param, notificationChan chan<- push.Event) {
	lp.notificationChan = notificationChan
	lp.uiChan = uiChan

	if lp.pvMeter == nil && lp.gridMeter == nil {
		log.FATAL.Fatal("missing either pv or grid meter")
	}

	// ensure charge meter exists
	if lp.chargeMeter == nil {
		if mt, ok := lp.charger.(api.Meter); ok {
			lp.chargeMeter = mt
		} else {
			mt := &wrapper.ChargeMeter{}
			_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentHandler)
			_ = lp.bus.Subscribe(evStopCharge, func() {
				mt.SetPower(0)
			})
			lp.chargeMeter = mt
		}
	}

	// ensure charge rater exists
	if rt, ok := lp.charger.(api.ChargeRater); ok {
		lp.chargeRater = rt
	} else {
		rt := wrapper.NewChargeRater(lp.Name, lp.chargeMeter)
		_ = lp.bus.Subscribe(evChargePower, rt.SetChargePower)
		_ = lp.bus.Subscribe(evStartCharge, rt.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, rt.StopCharge)
		lp.chargeRater = rt
	}

	// ensure charge timer exists
	if ct, ok := lp.charger.(api.ChargeTimer); ok {
		lp.chargeTimer = ct
	} else {
		ct := wrapper.NewChargeTimer()
		_ = lp.bus.Subscribe(evStartCharge, ct.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, ct.StopCharge)
		lp.chargeTimer = ct
	}

	// event handlers
	_ = lp.bus.Subscribe(evStartCharge, lp.evChargeStartHandler)
	_ = lp.bus.Subscribe(evStopCharge, lp.evChargeStopHandler)

	// read initial enabled state
	enabled, err := lp.charger.Enabled()
	if err == nil {
		lp.enabled = enabled
		log.INFO.Printf("%s charger %sd", lp.Name, status[lp.enabled])

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
	status, err := lp.charger.Status()
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

func (lp *LoadPoint) maxCurrent(mode api.ChargeMode) int64 {
	// grid meter will always be available, if as wrapped pv meter
	targetPower := lp.chargePower - lp.gridPower - lp.batteryPower - lp.ResidualPower
	if lp.batteryMeter == nil {
		log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW residual", lp.Name, targetPower, lp.chargePower, lp.gridPower, lp.ResidualPower)
	} else {
		log.DEBUG.Printf("%s target power: %.0fW = %.0fW charge - %.0fW grid - %.0fW battery - %.0fW residual", lp.Name, targetPower, lp.chargePower, lp.gridPower, lp.batteryPower, lp.ResidualPower)
	}

	// get max charge current
	targetCurrent := clamp(powerToCurrent(targetPower, lp.Voltage, lp.Phases), 0, lp.MaxCurrent)
	if targetCurrent < lp.MinCurrent {
		switch mode {
		case api.ModeMinPV:
			targetCurrent = lp.MinCurrent
		case api.ModePV:
			targetCurrent = 0
		}
	}

	return targetCurrent
}

// updateModePV handles "minpv" or "pv" modes by setting charger enabled/disabled state
// and maximum current according to available PV power
func (lp *LoadPoint) updateModePV(mode api.ChargeMode) error {
	targetCurrent := lp.maxCurrent(mode)
	if !lp.connected() {
		// ensure minimum current when not connected
		// https://github.com/andig/evcc/issues/105
		targetCurrent = min(lp.MinCurrent, targetCurrent)
	}

	log.DEBUG.Printf("%s target charge current: %dA", lp.Name, targetCurrent)

	if targetCurrent == 0 {
		return lp.rampOff()
	}

	if !lp.enabled {
		return lp.rampOn(targetCurrent)
	}

	return lp.rampUpDown(targetCurrent)
}

// updateMeter updates and publishes single meter
func (lp *LoadPoint) updateMeter(name string, meter api.Meter, power *float64) error {
	value, err := meter.CurrentPower()
	if err != nil {
		return err
	}

	*power = value // update value if no error

	log.DEBUG.Printf("%s %s power: %.1fW", lp.Name, name, *power)
	lp.publish(name+"Power", *power)

	return nil
}

// updateMeter updates and publishes single meter
func (lp *LoadPoint) updateMeters() (err error) {
	retryMeter := func(s string, m api.Meter, f *float64) {
		if m != nil {
			e := retry.Do(func() error {
				return lp.updateMeter(s, m, f)
			}, retry.Attempts(3))

			if e != nil {
				err = errors.Wrapf(e, "updating %s meter", s)
				log.ERROR.Printf("%s %v", lp.Name, err)
			}
		}
	}

	// read PV meter before charge meter
	retryMeter("grid", lp.gridMeter, &lp.gridPower)
	retryMeter("pv", lp.pvMeter, &lp.pvPower)
	retryMeter("battery", lp.batteryMeter, &lp.batteryPower)
	retryMeter("charge", lp.chargeMeter, &lp.chargePower)

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

	// execute loading strategy
	switch mode := lp.GetMode(); mode {
	case api.ModeOff:
		// apply immediately
		lp.guardUpdated = lp.clock.Now()
		err = lp.rampOff()
	case api.ModeNow:
		// apply immediately
		lp.guardUpdated = lp.clock.Now()
		// ensure that new connections happen at min current
		current := lp.MinCurrent
		if lp.connected() {
			current = lp.MaxCurrent
		}
		err = lp.rampOn(current)
	case api.ModeMinPV, api.ModePV:
		// pv modes require meter measurements
		if meterErr != nil {
			log.WARN.Printf("%s aborting due to meter error", lp.Name)
			break
		}
		err = lp.updateModePV(mode)
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
