package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/db"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"

	evbus "github.com/asaskevich/EventBus"
	"github.com/avast/retry-go/v3"
	"github.com/benbjohnson/clock"
	"github.com/cjrd/allocate"
	"github.com/emirpasic/gods/queues"
	aq "github.com/emirpasic/gods/queues/arrayqueue"
)

const (
	evChargeStart         = "start"      // update chargeTimer
	evChargeStop          = "stop"       // update chargeTimer
	evChargeCurrent       = "current"    // update fakeChargeMeter
	evChargePower         = "power"      // update chargeRater
	evVehicleConnect      = "connect"    // vehicle connected
	evVehicleDisconnect   = "disconnect" // vehicle disconnected
	evVehicleSoC          = "soc"        // vehicle soc progress
	evVehicleUnidentified = "guest"      // vehicle unidentified

	pvTimer   = "pv"
	pvEnable  = "enable"
	pvDisable = "disable"

	phaseTimer   = "phase"
	phaseScale1p = "scale1p"
	phaseScale3p = "scale3p"

	timerInactive = "inactive"

	minActiveCurrent      = 1.0 // minimum current at which a phase is treated as active
	vehicleDetectInterval = 1 * time.Minute
	vehicleDetectDuration = 10 * time.Minute

	guardGracePeriod = 10 * time.Second // allow out of sync during this timespan
)

// elapsed is the time an expired timer will be set to
var elapsed = time.Unix(0, 1)

// PollConfig defines the vehicle polling mode and interval
type PollConfig struct {
	Mode     string        `mapstructure:"mode"`     // polling mode charging (default), connected, always
	Interval time.Duration `mapstructure:"interval"` // interval when not charging
}

// SoCConfig defines soc settings, estimation and update behaviour
type SoCConfig struct {
	Poll     PollConfig `mapstructure:"poll"`
	Estimate *bool      `mapstructure:"estimate"`
	Min_     int        `mapstructure:"min"`    // TODO deprecated
	Target_  int        `mapstructure:"target"` // TODO deprecated
	min      int        // Default minimum SoC, guarded by mutex
	target   int        // Default target SoC, guarded by mutex
}

// Poll modes
const (
	pollCharging  = "charging"
	pollConnected = "connected"
	pollAlways    = "always"

	pollInterval = 60 * time.Minute
)

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

	Title             string   `mapstructure:"title"`    // UI title
	ConfiguredPhases  int      `mapstructure:"phases"`   // Charger configured phase mode 0/1/3
	ChargerRef        string   `mapstructure:"charger"`  // Charger reference
	VehicleRef        string   `mapstructure:"vehicle"`  // Vehicle reference
	VehiclesRef_      []string `mapstructure:"vehicles"` // TODO deprecated
	MeterRef          string   `mapstructure:"meter"`    // Charge meter reference
	SoC               SoCConfig
	Enable, Disable   ThresholdConfig
	ResetOnDisconnect bool `mapstructure:"resetOnDisconnect"`
	onDisconnect      api.ActionConfig
	targetEnergy      float64 // Target charge energy for dumb vehicles

	MinCurrent    float64       // PV mode: start current	Min+PV mode: min current
	MaxCurrent    float64       // Max allowed current. Physically ensured by the charger
	GuardDuration time.Duration // charger enable/disable minimum holding time

	enabled             bool      // Charger enabled state
	phases              int       // Charger enabled phases, guarded by mutex
	measuredPhases      int       // Charger physically measured phases
	chargeCurrent       float64   // Charger current limit
	guardUpdated        time.Time // Charger enabled/disabled timestamp
	socUpdated          time.Time // SoC updated timestamp (poll: connected)
	vehicleDetect       time.Time // Vehicle connected timestamp
	vehicleDetectTicker *clock.Ticker
	vehicleIdentifier   string

	charger     api.Charger
	chargeTimer api.ChargeTimer
	chargeRater api.ChargeRater

	chargeMeter    api.Meter   // Charger usage meter
	vehicle        api.Vehicle // Currently active vehicle
	defaultVehicle api.Vehicle // Default vehicle (disables detection)
	coordinator    coordinator.API
	socEstimator   *soc.Estimator
	socTimer       *soc.Timer

	// cached state
	status         api.ChargeStatus       // Charger status
	remoteDemand   loadpoint.RemoteDemand // External status demand
	chargePower    float64                // Charging power
	chargeCurrents []float64              // Phase currents
	connectedTime  time.Time              // Time when vehicle was connected
	pvTimer        time.Time              // PV enabled/disable timer
	phaseTimer     time.Time              // 1p3p switch timer
	wakeUpTimer    *Timer                 // Vehicle wake-up timeout

	// charge progress
	vehicleSoc              float64       // Vehicle SoC
	chargeDuration          time.Duration // Charge duration
	chargedEnergy           float64       // Charged energy while connected in Wh
	chargeRemainingDuration time.Duration // Remaining charge duration
	chargeRemainingEnergy   float64       // Remaining charge energy in Wh
	progress                *Progress     // Step-wise progress indicator

	// session log
	db      db.Database
	session *db.Session

	tasks queues.Queue // tasks to be executed
}

// NewLoadPointFromConfig creates a new loadpoint
func NewLoadPointFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) (*LoadPoint, error) {
	lp := NewLoadPoint(log)
	if err := util.DecodeOther(other, lp); err != nil {
		return nil, err
	}

	// set vehicle polling mode
	switch lp.SoC.Poll.Mode = strings.ToLower(lp.SoC.Poll.Mode); lp.SoC.Poll.Mode {
	case pollCharging:
	case pollConnected, pollAlways:
		lp.log.WARN.Printf("poll mode '%s' may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.SoC.Poll)
	default:
		if lp.SoC.Poll.Mode != "" {
			lp.log.WARN.Printf("invalid poll mode: %s", lp.SoC.Poll.Mode)
		}
		lp.SoC.Poll.Mode = pollCharging
	}

	// set vehicle polling interval
	if lp.SoC.Poll.Interval < pollInterval {
		if lp.SoC.Poll.Interval == 0 {
			lp.SoC.Poll.Interval = pollInterval
		} else {
			lp.log.WARN.Printf("poll interval '%v' is lower than %v and may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.SoC.Poll.Interval, pollInterval)
		}
	}

	if lp.MinCurrent == 0 {
		lp.log.WARN.Println("minCurrent must not be zero")
	}

	if lp.MaxCurrent < lp.MinCurrent {
		lp.log.WARN.Println("maxCurrent must be larger than minCurrent")
	}

	if lp.SoC.Min_ != 0 {
		lp.log.WARN.Println("Configuring soc.min at loadpoint is deprecated and must be applied per vehicle")
	}

	if lp.SoC.Target_ != 0 {
		lp.log.WARN.Println("Configuring soc.target at loadpoint is deprecated and must be applied per vehicle")
	}

	// store defaults
	lp.collectDefaults()

	if lp.MeterRef != "" {
		var err error
		if lp.chargeMeter, err = cp.Meter(lp.MeterRef); err != nil {
			return nil, err
		}
	}

	// default vehicle
	if lp.VehicleRef != "" {
		var err error
		if lp.defaultVehicle, err = cp.Vehicle(lp.VehicleRef); err != nil {
			return nil, err
		}
	}

	// TODO deprecated
	if len(lp.VehiclesRef_) > 0 {
		lp.log.WARN.Println("vehicles option is deprecated")
	}

	if lp.ChargerRef == "" {
		return nil, errors.New("missing charger")
	}
	var err error
	if lp.charger, err = cp.Charger(lp.ChargerRef); err != nil {
		return nil, err
	}
	lp.configureChargerType(lp.charger)

	// setup fixed phases:
	// - simple charger starts with phases config if specified or 3p
	// - switchable charger starts at 0p since we don't know the current setting
	if _, ok := lp.charger.(api.PhaseSwitcher); !ok {
		if lp.ConfiguredPhases == 0 {
			lp.ConfiguredPhases = 3
			lp.log.WARN.Println("phases not configured, assuming 3p")
		}
		lp.phases = lp.ConfiguredPhases
	} else if lp.ConfiguredPhases != 0 {
		lp.log.WARN.Printf("locking phase config to %dp for switchable charger", lp.ConfiguredPhases)
	}

	// validate thresholds
	if lp.Enable.Threshold > lp.Disable.Threshold {
		lp.log.WARN.Printf("PV mode enable threshold (%.0fW) is larger than disable threshold (%.0fW)", lp.Enable.Threshold, lp.Disable.Threshold)
	} else if lp.Enable.Threshold > 0 {
		lp.log.WARN.Printf("PV mode enable threshold %.0fW > 0 will start PV charging on grid power consumption. Did you mean -%.0f?", lp.Enable.Threshold, lp.Enable.Threshold)
	}

	return lp, nil
}

// NewLoadPoint creates a LoadPoint with sane defaults
func NewLoadPoint(log *util.Logger) *LoadPoint {
	clock := clock.New()
	bus := evbus.New()

	lp := &LoadPoint{
		log:           log,   // logger
		clock:         clock, // mockable time
		bus:           bus,   // event bus
		Mode:          api.ModeOff,
		status:        api.StatusNone,
		MinCurrent:    6,                                                     // A
		MaxCurrent:    16,                                                    // A
		SoC:           SoCConfig{min: 0, target: 100},                        // %
		Enable:        ThresholdConfig{Delay: time.Minute, Threshold: 0},     // t, W
		Disable:       ThresholdConfig{Delay: 3 * time.Minute, Threshold: 0}, // t, W
		GuardDuration: 5 * time.Minute,
		progress:      NewProgress(0, 10),     // soc progress indicator
		coordinator:   coordinator.NewDummy(), // dummy vehicle coordinator
		tasks:         aq.New(),               // task queue
	}

	// allow target charge handler to access loadpoint
	lp.socTimer = soc.NewTimer(lp.log, &adapter{LoadPoint: lp})

	return lp
}

// collectDefaults collects default values for use on disconnect
func (lp *LoadPoint) collectDefaults() {
	// get reference to action config
	actionCfg := &lp.onDisconnect

	// allocate action config such that all pointer fields are fully allocated
	if err := allocate.Zero(actionCfg); err == nil {
		// initialize with default values
		*actionCfg.Mode = lp.GetMode()
		*actionCfg.MinCurrent = lp.GetMinCurrent()
		*actionCfg.MaxCurrent = lp.GetMaxCurrent()
		*actionCfg.MinSoC = lp.GetMinSoC()
		*actionCfg.TargetSoC = lp.GetTargetSoC()
	} else {
		lp.log.ERROR.Printf("error allocating action config: %v", err)
	}
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
	var integrated bool

	// ensure charge meter exists
	if lp.chargeMeter == nil {
		integrated = true

		if mt, ok := charger.(api.Meter); ok {
			lp.chargeMeter = mt
		} else {
			mt := new(wrapper.ChargeMeter)
			_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentWrappedMeterHandler)
			_ = lp.bus.Subscribe(evChargeStop, func() { mt.SetPower(0) })
			lp.chargeMeter = mt
		}
	}

	// ensure charge rater exists
	// measurement are obtained from separate charge meter if defined
	// (https://github.com/evcc-io/evcc/issues/2469)
	if rt, ok := charger.(api.ChargeRater); ok && integrated {
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

	// add wakeup timer
	lp.wakeUpTimer = NewTimer()
}

// pushEvent sends push messages to clients
func (lp *LoadPoint) pushEvent(event string) {
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
	lp.pushEvent(evChargeStart)

	lp.wakeUpTimer.Stop()

	// soc update reset
	lp.socUpdated = time.Time{}

	// set created when first charging session segment starts
	lp.updateSession(func(session *db.Session) {
		if session.Created.IsZero() {
			session.Created = lp.clock.Now()
		}
	})
}

// evChargeStopHandler sends external stop event
func (lp *LoadPoint) evChargeStopHandler() {
	lp.log.INFO.Println("stop charging <-")
	lp.pushEvent(evChargeStop)

	// soc update reset
	lp.socUpdated = time.Time{}

	// reset pv enable/disable timer
	// https://github.com/evcc-io/evcc/issues/2289
	if !lp.pvTimer.Equal(elapsed) {
		lp.resetPVTimerIfRunning()
	}

	lp.stopSession()
}

// evVehicleConnectHandler sends external start event
func (lp *LoadPoint) evVehicleConnectHandler() {
	lp.log.INFO.Printf("car connected")

	// energy
	lp.setChargedEnergy(0)
	lp.publish("chargedEnergy", lp.getChargedEnergy())

	// duration
	lp.connectedTime = lp.clock.Now()
	lp.publish("connectedDuration", time.Duration(0))

	// soc update reset
	lp.socUpdated = time.Time{}

	// soc update reset on car change
	if lp.socEstimator != nil {
		lp.socEstimator.Reset()
	}

	// set default or start detection
	lp.vehicleDefaultOrDetect()

	// immediately allow pv mode activity
	lp.elapsePVTimer()

	// create charging session
	lp.createSession()
}

// evVehicleDisconnectHandler sends external start event
func (lp *LoadPoint) evVehicleDisconnectHandler() {
	lp.log.INFO.Println("car disconnected")

	// session is persisted during evChargeStopHandler which runs before
	lp.clearSession()

	// phases are unknown when vehicle disconnects
	lp.resetMeasuredPhases()

	// energy and duration
	lp.publish("chargedEnergy", lp.getChargedEnergy())
	lp.publish("connectedDuration", lp.clock.Since(lp.connectedTime))

	// remove charger vehicle id and stop potential detection
	lp.setVehicleIdentifier("")
	lp.stopVehicleDetection()

	// remove active vehicle if not default
	if lp.vehicle != lp.defaultVehicle {
		lp.setActiveVehicle(lp.defaultVehicle)
	}

	// set default mode on disconnect
	if lp.ResetOnDisconnect {
		actionCfg := lp.onDisconnect
		if v := lp.defaultVehicle; v != nil {
			actionCfg = actionCfg.Merge(v.OnIdentified())
		}
		lp.applyAction(actionCfg)
	}

	// soc update reset
	lp.socUpdated = time.Time{}

	// reset timer when vehicle is removed
	lp.socTimer.Reset()
}

// evVehicleSoCProgressHandler sends external start event
func (lp *LoadPoint) evVehicleSoCProgressHandler(soc float64) {
	if lp.progress.NextStep(soc) {
		lp.pushEvent(evVehicleSoC)
	}
}

// evChargeCurrentHandler publishes the charge current
func (lp *LoadPoint) evChargeCurrentHandler(current float64) {
	if !lp.enabled {
		current = 0
	}
	lp.publish("chargeCurrent", current)
}

// evChargeCurrentWrappedMeterHandler updates the dummy charge meter's charge power.
// This simplifies the main flow where the charge meter can always be treated as present.
// It assumes that the charge meter cannot consume more than total household consumption.
// If physical charge meter is present this handler is not used.
// The actual value is published by the evChargeCurrentHandler
func (lp *LoadPoint) evChargeCurrentWrappedMeterHandler(current float64) {
	power := current * float64(lp.activePhases()) * Voltage

	// if disabled we cannot be charging
	if !lp.enabled || !lp.charging() {
		power = 0
	}

	// handler only called if charge meter was replaced by dummy
	lp.chargeMeter.(*wrapper.ChargeMeter).SetPower(power)
}

// applyAction executes the action
func (lp *LoadPoint) applyAction(actionCfg api.ActionConfig) {
	if actionCfg.Mode != nil {
		lp.SetMode(*actionCfg.Mode)
	}
	if min := actionCfg.MinCurrent; min != nil && *min >= *lp.onDisconnect.MinCurrent {
		lp.SetMinCurrent(*min)
	}
	if max := actionCfg.MaxCurrent; max != nil && *max <= *lp.onDisconnect.MaxCurrent {
		lp.SetMaxCurrent(*max)
	}
	if actionCfg.MinSoC != nil {
		lp.SetMinSoC(*actionCfg.MinSoC)
	}
	if actionCfg.TargetSoC != nil {
		lp.SetTargetSoC(*actionCfg.TargetSoC)
	}
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
	_ = lp.bus.Subscribe(evVehicleSoC, lp.evVehicleSoCProgressHandler)

	// publish initial values
	lp.publish("title", lp.Title)
	lp.publish("minCurrent", lp.MinCurrent)
	lp.publish("maxCurrent", lp.MaxCurrent)

	lp.setConfiguredPhases(lp.ConfiguredPhases)
	lp.publish(phasesEnabled, lp.phases)
	lp.publish(phasesActive, lp.activePhases())
	lp.publishTimer(phaseTimer, 0, timerInactive)
	lp.publishTimer(pvTimer, 0, timerInactive)

	// assign and publish default vehicle
	if lp.defaultVehicle != nil {
		lp.setActiveVehicle(lp.defaultVehicle)
	}

	lp.Lock()
	lp.publish("mode", lp.Mode)
	lp.publish("targetSoC", lp.SoC.target)
	lp.publish("minSoC", lp.SoC.min)
	lp.Unlock()

	// reset detection state
	lp.publish(vehicleDetectionActive, false)

	// read initial charger state to prevent immediately disabling charger
	if enabled, err := lp.charger.Enabled(); err == nil {
		if lp.enabled = enabled; enabled {
			lp.guardUpdated = lp.clock.Now()
			// set defined current for use by pv mode
			_ = lp.setLimit(lp.GetMinCurrent(), false)
		}
	} else {
		lp.log.ERROR.Printf("charger: %v", err)
	}

	// allow charger to access loadpoint
	if ctrl, ok := lp.charger.(loadpoint.Controller); ok {
		ctrl.LoadpointControl(lp)
	}
}

// syncCharger updates charger status and synchronizes it with expectations
func (lp *LoadPoint) syncCharger() {
	enabled, err := lp.charger.Enabled()
	if err == nil {
		if enabled != lp.enabled {
			if time.Since(lp.guardUpdated) > guardGracePeriod {
				lp.log.WARN.Printf("charger out of sync: expected %vd, got %vd", status[lp.enabled], status[enabled])
			}
			err = lp.charger.Enable(lp.enabled)
		}

		if !enabled && lp.charging() {
			lp.log.WARN.Println("charger logic error: disabled but charging")
		}
	}

	if err != nil {
		lp.log.ERROR.Printf("charger: %v", err)
	}
}

// setLimit applies charger current limits and enables/disables accordingly
func (lp *LoadPoint) setLimit(chargeCurrent float64, force bool) error {
	// set current
	if chargeCurrent != lp.chargeCurrent && chargeCurrent >= lp.GetMinCurrent() {
		var err error
		if charger, ok := lp.charger.(api.ChargerEx); ok && !lp.vehicleHasFeature(api.CoarseCurrent) {
			err = charger.MaxCurrentMillis(chargeCurrent)
		} else {
			chargeCurrent = math.Trunc(chargeCurrent)
			err = lp.charger.MaxCurrent(int64(chargeCurrent))
		}

		if err != nil {
			return fmt.Errorf("max charge current %.3gA: %w", chargeCurrent, err)
		}

		lp.log.DEBUG.Printf("max charge current: %.3gA", chargeCurrent)
		lp.chargeCurrent = chargeCurrent
		lp.bus.Publish(evChargeCurrent, chargeCurrent)
	}

	// set enabled/disabled
	if enabled := chargeCurrent >= lp.GetMinCurrent(); enabled != lp.enabled {
		if remaining := (lp.GuardDuration - lp.clock.Since(lp.guardUpdated)).Truncate(time.Second); remaining > 0 && !force {
			lp.log.DEBUG.Printf("charger %s: contactor delay %v", status[enabled], remaining)
			return nil
		}

		// remote stop
		// TODO https://github.com/evcc-io/evcc/discussions/1929
		// if car, ok := lp.vehicle.(api.VehicleChargeController); !enabled && ok {
		// 	// log but don't propagate
		// 	if err := car.StopCharge(); err != nil {
		// 		lp.log.ERROR.Printf("vehicle remote charge stop: %v", err)
		// 	}
		// }

		if err := lp.charger.Enable(enabled); err != nil {
			return fmt.Errorf("charger %s: %w", status[enabled], err)
		}

		lp.log.DEBUG.Printf("charger %s", status[enabled])
		lp.enabled = enabled
		lp.guardUpdated = lp.clock.Now()

		lp.bus.Publish(evChargeCurrent, chargeCurrent)

		// start/stop vehicle wake-up timer
		if enabled {
			lp.log.DEBUG.Printf("wake-up timer: start")
			lp.wakeUpTimer.Start()
		} else {
			lp.log.DEBUG.Printf("wake-up timer: stop")
			lp.wakeUpTimer.Stop()
		}

		// remote start
		// TODO https://github.com/evcc-io/evcc/discussions/1929
		// if car, ok := lp.vehicle.(api.VehicleChargeController); enabled && ok {
		// 	// log but don't propagate
		// 	if err := car.StartCharge(); err != nil {
		// 		lp.log.ERROR.Printf("vehicle remote charge start: %v", err)
		// 	}
		// }
	}

	return nil
}

// connected returns the EVs connection state
func (lp *LoadPoint) connected() bool {
	status := lp.GetStatus()
	return status == api.StatusB || status == api.StatusC
}

// charging returns the EVs charging state
func (lp *LoadPoint) charging() bool {
	return lp.GetStatus() == api.StatusC
}

// charging returns the EVs charging state
func (lp *LoadPoint) setStatus(status api.ChargeStatus) {
	lp.Lock()
	defer lp.Unlock()
	lp.status = status
}

// targetEnergyReached checks if target is configured and reached
func (lp *LoadPoint) targetEnergyReached() bool {
	return (lp.vehicle == nil || lp.vehicleHasFeature(api.Offline)) &&
		lp.targetEnergy > 0 &&
		lp.getChargedEnergy()/1e3 >= float64(lp.targetEnergy)
}

// targetSocReached checks if target is configured and reached.
// If vehicle is not configured this will always return false
func (lp *LoadPoint) targetSocReached() bool {
	return lp.vehicle != nil &&
		lp.SoC.target > 0 &&
		lp.SoC.target < 100 &&
		lp.vehicleSoc >= float64(lp.SoC.target)
}

// minSocNotReached checks if minimum is configured and not reached.
// If vehicle is not configured this will always return true
func (lp *LoadPoint) minSocNotReached() bool {
	return lp.vehicle != nil &&
		lp.SoC.min > 0 &&
		lp.vehicleSoc < float64(lp.SoC.min)
}

// climateActive checks if vehicle has active climate request
func (lp *LoadPoint) climateActive() bool {
	if cl, ok := lp.vehicle.(api.VehicleClimater); ok {
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

		if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("climater: %v", err)
		}
	}

	return false
}

// disableUnlessClimater disables the charger unless climate is active
func (lp *LoadPoint) disableUnlessClimater() error {
	var current float64 // zero disables
	if lp.climateActive() {
		lp.log.DEBUG.Println("climater active")
		current = lp.GetMinCurrent()
	}
	lp.socTimer.Reset() // once SoC is reached, the target charge request is removed
	return lp.setLimit(current, true)
}

// remoteControlled returns true if remote control status is active
func (lp *LoadPoint) remoteControlled(demand loadpoint.RemoteDemand) bool {
	lp.Lock()
	defer lp.Unlock()

	return lp.remoteDemand == demand
}

// setVehicleIdentifier updated the vehicle id as read from the charger
func (lp *LoadPoint) setVehicleIdentifier(id string) {
	if lp.vehicleIdentifier != id {
		lp.vehicleIdentifier = id
		lp.publish("vehicleIdentity", id)
	}
}

// identifyVehicle reads vehicle identification from charger
func (lp *LoadPoint) identifyVehicle() {
	identifier, ok := lp.charger.(api.Identifier)
	if !ok {
		return
	}

	id, err := identifier.Identify()
	if err != nil {
		lp.log.ERROR.Println("charger vehicle id:", err)
		return
	}

	if lp.vehicleIdentifier == id {
		return
	}

	// vehicle found or removed
	lp.setVehicleIdentifier(id)

	if id != "" {
		lp.log.DEBUG.Println("charger vehicle id:", id)

		if vehicle := lp.selectVehicleByID(id); vehicle != nil {
			lp.stopVehicleDetection()
			lp.setActiveVehicle(vehicle)
		}
	}
}

// selectVehicleByID selects the vehicle with the given ID
func (lp *LoadPoint) selectVehicleByID(id string) api.Vehicle {
	vehicles := lp.coordinatedVehicles()

	// find exact match
	for _, vehicle := range vehicles {
		for _, vid := range vehicle.Identifiers() {
			if strings.EqualFold(id, vid) {
				return vehicle
			}
		}
	}

	// find placeholder match
	for _, vehicle := range vehicles {
		for _, vid := range vehicle.Identifiers() {
			// case insensitive match
			re, err := regexp.Compile("(?i)" + strings.ReplaceAll(vid, "*", ".*?"))
			if err != nil {
				lp.log.ERROR.Printf("vehicle id: %v", err)
				continue
			}

			if re.MatchString(id) {
				return vehicle
			}
		}
	}

	return nil
}

// setActiveVehicle assigns currently active vehicle, configures soc estimator
// and adds an odometer task
func (lp *LoadPoint) setActiveVehicle(vehicle api.Vehicle) {
	lp.Lock()
	defer lp.Unlock()

	if lp.vehicle == vehicle {
		return
	}

	from := "unknown"
	if lp.vehicle != nil {
		lp.coordinator.Release(lp.vehicle)
		from = lp.vehicle.Title()
	}
	to := "unknown"
	if vehicle != nil {
		lp.coordinator.Acquire(vehicle)
		to = vehicle.Title()
	}
	lp.log.INFO.Printf("vehicle updated: %s -> %s", from, to)

	// reset minSoC and targetSoC before change
	lp.setMinSoC(0)
	lp.setTargetSoC(100)

	if lp.vehicle = vehicle; vehicle != nil {
		lp.socUpdated = time.Time{}

		// resolve optional config
		var estimate bool
		if lp.SoC.Estimate == nil || *lp.SoC.Estimate {
			estimate = true
		}
		lp.socEstimator = soc.NewEstimator(lp.log, lp.charger, vehicle, estimate)

		lp.publish("vehiclePresent", true)
		lp.publish("vehicleTitle", lp.vehicle.Title())
		lp.publish("vehicleIcon", lp.vehicle.Icon())
		lp.publish("vehicleCapacity", lp.vehicle.Capacity())

		// unblock api
		lp.Unlock()
		lp.applyAction(vehicle.OnIdentified())
		lp.Lock()

		lp.addTask(lp.vehicleOdometer)

		lp.progress.Reset()
	} else {
		lp.socEstimator = nil

		lp.publish("vehiclePresent", false)
		lp.publish("vehicleTitle", "")
		lp.publish("vehicleIcon", "")
		lp.publish("vehicleCapacity", int64(0))
		lp.publish(vehicleOdometer, 0.0)
	}

	// reset target energy
	lp.setTargetEnergy(0)

	// re-publish vehicle settings
	lp.Unlock()
	lp.publish(phasesActive, lp.activePhases())
	lp.Lock()

	lp.unpublishVehicle()

	lp.updateSession(func(session *db.Session) {
		var title string
		if lp.vehicle != nil {
			title = lp.vehicle.Title()
		}

		lp.session.Vehicle = title
	})
}

func (lp *LoadPoint) wakeUpVehicle() {
	// charger
	if c, ok := lp.charger.(api.Resurrector); ok {
		if err := c.WakeUp(); err != nil {
			lp.log.ERROR.Printf("wake-up charger: %v", err)
		}
		return
	}

	// vehicle
	if vs, ok := lp.vehicle.(api.Resurrector); ok {
		if err := vs.WakeUp(); err != nil {
			lp.log.ERROR.Printf("wake-up vehicle: %v", err)
		}
	}
}

// unpublishVehicle resets published vehicle data
func (lp *LoadPoint) unpublishVehicle() {
	lp.vehicleSoc = 0

	lp.publish("vehicleSoC", 0.0)
	lp.publish(vehicleRange, int64(0))
	lp.publish(vehicleTargetSoC, 0.0)

	lp.setRemainingDuration(-1)

	lp.vehiclePublishFeature(api.Offline)
}

// vehicleHasFeature checks availability of vehicle feature
func (lp *LoadPoint) vehicleHasFeature(f api.Feature) bool {
	v, ok := lp.vehicle.(api.FeatureDescriber)
	if ok {
		ok = v.Has(f)
	}
	return ok
}

// vehiclePublishFeature availability of vehicle features
func (lp *LoadPoint) vehiclePublishFeature(f api.Feature) {
	lp.publish("vehicleFeature"+f.String(), lp.vehicleHasFeature(f))
}

// vehicleUnidentified returns true if there are associated vehicles and detection is running.
// It will also reset the api cache at regular intervals.
// Detection is stopped after maximum duration and the "guest vehicle" message dispatched.
func (lp *LoadPoint) vehicleUnidentified() bool {
	if lp.vehicle != nil || lp.vehicleDetect.IsZero() || len(lp.coordinatedVehicles()) == 0 {
		return false
	}

	// stop detection
	if lp.clock.Since(lp.vehicleDetect) > vehicleDetectDuration {
		lp.stopVehicleDetection()
		lp.pushEvent(evVehicleUnidentified)
		return false
	}

	// request vehicle api refresh while waiting to identify
	select {
	case <-lp.vehicleDetectTicker.C:
		lp.log.DEBUG.Println("vehicle api refresh")
		provider.ResetCached()
	default:
	}

	return true
}

// vehicleDefaultOrDetect will assign and update default vehicle or start detection
func (lp *LoadPoint) vehicleDefaultOrDetect() {
	if lp.defaultVehicle != nil {
		if lp.vehicle != lp.defaultVehicle {
			lp.setActiveVehicle(lp.defaultVehicle)
		} else {
			// default vehicle is already active, update odometer anyway
			// need to do this here since setActiveVehicle would short-circuit
			lp.addTask(lp.vehicleOdometer)
		}
	} else if len(lp.coordinatedVehicles()) > 0 && lp.connected() {
		lp.startVehicleDetection()
	}
}

// startVehicleDetection reset connection timer and starts api refresh timer
func (lp *LoadPoint) startVehicleDetection() {
	// flush all vehicles before detection starts
	lp.log.DEBUG.Println("vehicle api refresh")
	provider.ResetCached()

	lp.vehicleDetect = lp.clock.Now()
	lp.vehicleDetectTicker = lp.clock.Ticker(vehicleDetectInterval)
	lp.publish(vehicleDetectionActive, true)
}

// stopVehicleDetection expires the connection timer and ticker
func (lp *LoadPoint) stopVehicleDetection() {
	lp.vehicleDetect = time.Time{}
	if lp.vehicleDetectTicker != nil {
		lp.vehicleDetectTicker.Stop()
	}
	lp.publish(vehicleDetectionActive, false)
}

// identifyVehicleByStatus validates if the active vehicle is still connected to the loadpoint
func (lp *LoadPoint) identifyVehicleByStatus() {
	if len(lp.coordinatedVehicles()) == 0 {
		return
	}

	_, ok := lp.charger.(api.Identifier)

	if vehicle := lp.coordinator.IdentifyVehicleByStatus(!ok); vehicle != nil {
		lp.stopVehicleDetection()
		lp.setActiveVehicle(vehicle)
		return
	}

	// remove previous vehicle if status was not confirmed
	if _, ok := lp.vehicle.(api.ChargeState); ok {
		lp.setActiveVehicle(nil)
	}
}

// vehicleOdometer updates odometer
func (lp *LoadPoint) vehicleOdometer() {
	if vs, ok := lp.vehicle.(api.VehicleOdometer); ok {
		if odo, err := vs.Odometer(); err == nil {
			lp.log.DEBUG.Printf("vehicle odometer: %.0fkm", odo)
			lp.publish(vehicleOdometer, odo)

			// update session once odometer is read
			lp.updateSession(func(session *db.Session) {
				session.Odometer = odo
			})
		} else {
			lp.log.ERROR.Printf("vehicle odometer: %v", err)
		}
	}
}

// statusEvents converts the observed charger status change into a logical sequence of events
func statusEvents(prevStatus, status api.ChargeStatus) []string {
	res := make([]string, 0, 2)

	// changed from A - connected
	if prevStatus == api.StatusA || (status != api.StatusA && prevStatus == api.StatusNone) {
		res = append(res, evVehicleConnect)
	}

	// changed to C - start charging
	if status == api.StatusC {
		res = append(res, evChargeStart)
	}

	// changed from C - stop charging
	if prevStatus == api.StatusC {
		res = append(res, evChargeStop)
	}

	// changed to A - disconnected
	if status == api.StatusA {
		res = append(res, evVehicleDisconnect)
	}

	return res
}

// updateChargerStatus updates charger status and detects car connected/disconnected events
func (lp *LoadPoint) updateChargerStatus() error {
	status, err := lp.charger.Status()
	if err != nil {
		return err
	}

	lp.log.DEBUG.Printf("charger status: %s", status)

	if prevStatus := lp.GetStatus(); status != prevStatus {
		lp.setStatus(status)

		for _, ev := range statusEvents(prevStatus, status) {
			lp.bus.Publish(ev)

			// send connect/disconnect events except during startup
			if prevStatus != api.StatusNone {
				switch ev {
				case evVehicleConnect:
					lp.pushEvent(evVehicleConnect)
				case evVehicleDisconnect:
					lp.pushEvent(evVehicleDisconnect)
				}
			}
		}

		// update whenever there is a state change
		lp.bus.Publish(evChargeCurrent, lp.chargeCurrent)
	}

	return nil
}

// effectiveCurrent returns the currently effective charging current
func (lp *LoadPoint) effectiveCurrent() float64 {
	if !lp.charging() {
		return 0
	}

	// adjust actual current for vehicles like Zoe where it remains below target
	if lp.chargeCurrents != nil {
		cur := lp.chargeCurrents[0]
		return math.Min(cur+2.0, lp.chargeCurrent)
	}

	return lp.chargeCurrent
}

// elapsePVTimer puts the pv enable/disable timer into elapsed state
func (lp *LoadPoint) elapsePVTimer() {
	lp.log.DEBUG.Printf("pv timer elapse")

	lp.pvTimer = elapsed
	lp.guardUpdated = elapsed

	lp.publishTimer(pvTimer, 0, timerInactive)
}

// resetPVTimerIfRunning resets the pv enable/disable timer to disabled state
func (lp *LoadPoint) resetPVTimerIfRunning(typ ...string) {
	if lp.pvTimer.IsZero() {
		return
	}

	msg := "pv timer reset"
	if len(typ) == 1 {
		msg = fmt.Sprintf("pv %s timer reset", typ[0])
	}
	lp.log.DEBUG.Printf(msg)

	lp.pvTimer = time.Time{}
	lp.publishTimer(pvTimer, 0, timerInactive)
}

// resetPhaseTimer resets the phase switch timer to disabled state
func (lp *LoadPoint) resetPhaseTimer() {
	lp.phaseTimer = time.Time{}
	lp.publishTimer(phaseTimer, 0, timerInactive)
}

// scalePhasesRequired validates if fixed phase configuration matches enabled phases
func (lp *LoadPoint) scalePhasesRequired() bool {
	_, ok := lp.charger.(api.PhaseSwitcher)
	return ok && lp.ConfiguredPhases != 0 && lp.ConfiguredPhases != lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available
func (lp *LoadPoint) scalePhasesIfAvailable(phases int) error {
	if lp.ConfiguredPhases != 0 {
		phases = lp.ConfiguredPhases
	}

	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		return lp.scalePhases(phases)
	}

	return nil
}

// setConfiguredPhases sets the default phase configuration
func (lp *LoadPoint) setConfiguredPhases(phases int) {
	lp.Lock()
	defer lp.Unlock()

	lp.ConfiguredPhases = phases

	// publish 1p3p capability and phase configuration
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		lp.publish(phasesConfigured, lp.ConfiguredPhases)
	} else {
		lp.publish(phasesConfigured, nil)
	}
}

// setPhases sets the number of enabled phases without modifying the charger
func (lp *LoadPoint) setPhases(phases int) {
	if lp.GetPhases() != phases {
		lp.Lock()
		lp.phases = phases
		lp.Unlock()

		// reset timer to disabled state
		lp.resetPhaseTimer()

		// measure phases after switching
		lp.resetMeasuredPhases()
	}
}

// scalePhases adjusts the number of active phases and returns the appropriate charging current.
// Returns api.ErrNotAvailable if api.PhaseSwitcher is not available.
func (lp *LoadPoint) scalePhases(phases int) error {
	cp, ok := lp.charger.(api.PhaseSwitcher)
	if !ok {
		panic("charger does not implement api.PhaseSwitcher")
	}

	if lp.GetPhases() != phases {
		// disable charger - this will also stop the car charging using the api if available
		if err := lp.setLimit(0, true); err != nil {
			return err
		}

		// switch phases
		if err := cp.Phases1p3p(phases); err != nil {
			return fmt.Errorf("switch phases: %w", err)
		}

		// update setting and reset timer
		lp.setPhases(phases)

		// allow pv mode to re-enable charger right away
		lp.elapsePVTimer()
	}

	return nil
}

// pvScalePhases switches phases if necessary and returns if switch occurred
func (lp *LoadPoint) pvScalePhases(availablePower, minCurrent, maxCurrent float64) bool {
	phases := lp.GetPhases()

	// observed phase state inconsistency
	// - https://github.com/evcc-io/evcc/issues/1572
	// - https://github.com/evcc-io/evcc/issues/2230
	// - https://github.com/evcc-io/evcc/issues/2613
	measuredPhases := lp.getMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
	}

	var waiting bool
	activePhases := lp.activePhases()

	// scale down phases
	if targetCurrent := powerToCurrent(availablePower, activePhases); targetCurrent < minCurrent && activePhases > 1 && lp.ConfiguredPhases < 3 {
		lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.Disable.Delay, phaseScale1p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.Disable.Delay {
			lp.log.DEBUG.Printf("phase %s timer elapsed", phaseScale1p)
			if err := lp.scalePhases(1); err == nil {
				lp.log.DEBUG.Printf("switched phases: 1p @ %.0fW", availablePower)
			} else {
				lp.log.ERROR.Println(err)
			}
			return true
		}

		waiting = true
	}

	maxPhases := lp.maxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)
	scalable := maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, 3*Voltage*minCurrent, maxPhases)

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.Enable.Delay, phaseScale3p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.Enable.Delay {
			lp.log.DEBUG.Printf("phase %s timer elapsed", phaseScale3p)
			if err := lp.scalePhases(3); err == nil {
				lp.log.DEBUG.Printf("switched phases: 3p @ %.0fW", availablePower)
			} else {
				lp.log.ERROR.Println(err)
			}
			return true
		}

		waiting = true
	}

	// reset timer to disabled state
	if !waiting && !lp.phaseTimer.IsZero() {
		lp.resetPhaseTimer()
	}

	return false
}

// coordinatedVehicles is the slice of vehicles from the coordinator
func (lp *LoadPoint) coordinatedVehicles() []api.Vehicle {
	if lp.coordinator == nil {
		return nil
	}
	return lp.coordinator.GetVehicles()
}

// TODO move up to timer functions
func (lp *LoadPoint) publishTimer(name string, delay time.Duration, action string) {
	timer := lp.pvTimer
	if name == phaseTimer {
		timer = lp.phaseTimer
	}

	remaining := delay - lp.clock.Since(timer)
	if remaining < 0 {
		remaining = 0
	}

	lp.publish(name+"Action", action)
	lp.publish(name+"Remaining", remaining)

	if action == timerInactive {
		lp.log.DEBUG.Printf("%s timer %s", name, action)
	} else {
		lp.log.DEBUG.Printf("%s %s in %v", name, action, remaining.Round(time.Second))
	}
}

// pvMaxCurrent calculates the maximum target current for PV mode
func (lp *LoadPoint) pvMaxCurrent(mode api.ChargeMode, sitePower float64, batteryBuffered bool) float64 {
	// read only once to simplify testing
	minCurrent := lp.GetMinCurrent()
	maxCurrent := lp.GetMaxCurrent()

	// switch phases up/down
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		availablePower := -sitePower + lp.chargePower

		// in case of scaling, keep charger disabled for this cycle
		if lp.pvScalePhases(availablePower, minCurrent, maxCurrent) {
			return 0
		}
	}

	// calculate target charge current from delta power and actual current
	effectiveCurrent := lp.effectiveCurrent()
	activePhases := lp.activePhases()
	deltaCurrent := powerToCurrent(-sitePower, activePhases)
	targetCurrent := math.Max(effectiveCurrent+deltaCurrent, 0)

	lp.log.DEBUG.Printf("pv charge current: %.3gA = %.3gA + %.3gA (%.0fW @ %dp)", targetCurrent, effectiveCurrent, deltaCurrent, sitePower, activePhases)

	// in MinPV mode or under special conditions return at least minCurrent
	if (mode == api.ModeMinPV || batteryBuffered || lp.climateActive()) && targetCurrent < minCurrent {
		return minCurrent
	}

	if mode == api.ModePV && lp.enabled && targetCurrent < minCurrent {
		// kick off disable sequence
		if sitePower >= lp.Disable.Threshold && lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("site power %.0fW >= %.0fW disable threshold", sitePower, lp.Disable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv disable timer start: %v", lp.Disable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.Disable.Delay, pvDisable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Disable.Delay {
				lp.log.DEBUG.Println("pv disable timer elapsed")
				return 0
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv disable timer remaining: %v", (lp.Disable.Delay - elapsed).Round(time.Second))
			}
		} else {
			// reset timer
			lp.resetPVTimerIfRunning("disable")
		}

		// lp.log.DEBUG.Println("pv disable timer: keep enabled")
		return minCurrent
	}

	if mode == api.ModePV && !lp.enabled {
		// kick off enable sequence
		if (lp.Enable.Threshold == 0 && targetCurrent >= minCurrent) ||
			(lp.Enable.Threshold != 0 && sitePower <= lp.Enable.Threshold) {
			lp.log.DEBUG.Printf("site power %.0fW <= %.0fW enable threshold", sitePower, lp.Enable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv enable timer start: %v", lp.Enable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.Enable.Delay, pvEnable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Enable.Delay {
				lp.log.DEBUG.Println("pv enable timer elapsed")
				return minCurrent
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv enable timer remaining: %v", (lp.Enable.Delay - elapsed).Round(time.Second))
			}
		} else {
			// reset timer
			lp.resetPVTimerIfRunning("enable")
		}

		// lp.log.DEBUG.Println("pv enable timer: keep disabled")
		return 0
	}

	// reset timer to disabled state
	lp.resetPVTimerIfRunning()

	// cap at maximum current
	targetCurrent = math.Min(targetCurrent, maxCurrent)

	return targetCurrent
}

// UpdateChargePower updates charge meter power
func (lp *LoadPoint) UpdateChargePower() {
	err := retry.Do(func() error {
		value, err := lp.chargeMeter.CurrentPower()
		if err != nil {
			return err
		}

		lp.Lock()
		lp.chargePower = value // update value if no error
		lp.Unlock()

		lp.log.DEBUG.Printf("charge power: %.0fW", value)
		lp.publish("chargePower", value)

		// use -1 for https://github.com/evcc-io/evcc/issues/2153
		if lp.chargePower < -1 {
			lp.log.WARN.Printf("charge power must not be negative: %.0f", lp.chargePower)
		}

		return nil
	}, retryOptions...)
	if err != nil {
		lp.log.ERROR.Printf("charge meter: %v", err)
	}
}

// updateChargeCurrents uses MeterCurrent interface to count phases with current >=1A
func (lp *LoadPoint) updateChargeCurrents() {
	lp.chargeCurrents = nil

	phaseMeter, ok := lp.chargeMeter.(api.MeterCurrent)
	if !ok {
		return // don't guess
	}

	i1, i2, i3, err := phaseMeter.Currents()
	if err != nil {
		lp.log.ERROR.Printf("charge meter: %v", err)
		return
	}

	lp.chargeCurrents = []float64{i1, i2, i3}
	lp.log.DEBUG.Printf("charge currents: %.3gA", lp.chargeCurrents)
	lp.publish("chargeCurrents", lp.chargeCurrents)

	if lp.charging() {
		// Quine-McCluskey for (¬L1∧L2∧¬L3) ∨ (¬L1∧¬L2∧L3) ∨ (L1∧¬L2∧L3) ∨ (¬L1∧L2∧L3) -> ¬L1 ∧ L2 ∨ ¬L2 ∧ L3
		if !(i1 > minActiveCurrent) && (i2 > minActiveCurrent) || !(i2 > minActiveCurrent) && (i3 > minActiveCurrent) {
			lp.log.WARN.Printf("invalid phase wiring between charge meter and vehicle")
		}

		var phases int
		for _, i := range lp.chargeCurrents {
			if i > minActiveCurrent {
				phases++
			}
		}

		if phases >= 1 {
			lp.Lock()
			lp.measuredPhases = phases
			lp.Unlock()

			lp.log.DEBUG.Printf("detected phases: %dp", phases)
			lp.publish(phasesActive, phases)
		}
	}
}

// publish charged energy and duration
func (lp *LoadPoint) publishChargeProgress() {
	if f, err := lp.chargeRater.ChargedEnergy(); err == nil {
		// workaround for Go-E resetting during disconnect, see
		// https://github.com/evcc-io/evcc/issues/5092
		if f > 0 {
			lp.setChargedEnergy(1e3 * f) // convert to Wh
		}
	} else {
		lp.log.ERROR.Printf("charge rater: %v", err)
	}

	if d, err := lp.chargeTimer.ChargingTime(); err == nil {
		lp.chargeDuration = d.Round(time.Second)
	} else {
		lp.log.ERROR.Printf("charge timer: %v", err)
	}

	lp.publish("chargedEnergy", lp.getChargedEnergy())
	lp.publish("chargeDuration", lp.chargeDuration)
	if _, ok := lp.chargeMeter.(api.MeterEnergy); ok {
		lp.publish("chargeTotalImport", lp.chargeMeterTotal())
	}
}

// socPollAllowed validates charging state against polling mode
func (lp *LoadPoint) socPollAllowed() bool {
	remaining := lp.SoC.Poll.Interval - lp.clock.Since(lp.socUpdated)

	honourUpdateInterval := lp.SoC.Poll.Mode == pollAlways ||
		lp.SoC.Poll.Mode == pollConnected && lp.connected() ||
		lp.SoC.Poll.Mode == pollCharging && lp.connected() && (lp.vehicleSoc < float64(lp.SoC.target))

	if honourUpdateInterval && remaining > 0 {
		lp.log.DEBUG.Printf("next soc poll remaining time: %v", remaining.Truncate(time.Second))
	}

	return lp.charging() || honourUpdateInterval && (remaining <= 0) || lp.connected() && lp.socUpdated.IsZero()
}

// checks if the connected charger can provide SoC to the connected vehicle
func (lp *LoadPoint) socProvidedByCharger() bool {
	if charger, ok := lp.charger.(api.Battery); ok {
		if _, err := charger.SoC(); err == nil {
			return true
		}
	}
	return false
}

// publish state of charge, remaining charge duration and range
func (lp *LoadPoint) publishSoCAndRange() {
	if lp.socEstimator == nil {
		return
	}

	if lp.socPollAllowed() || lp.socProvidedByCharger() {
		var f float64
		var err error

		// guard for socEstimator removed by api
		if se := lp.socEstimator; se != nil {
			lp.socUpdated = lp.clock.Now()
			f, err = se.SoC(lp.getChargedEnergy())
		} else {
			return
		}

		if err != nil {
			if errors.Is(err, api.ErrMustRetry) {
				lp.socUpdated = time.Time{}
			} else {
				lp.log.ERROR.Printf("vehicle soc: %v", err)
			}

			return
		}

		lp.vehicleSoc = math.Trunc(f)
		lp.log.DEBUG.Printf("vehicle soc: %.0f%%", lp.vehicleSoc)
		lp.publish("vehicleSoC", lp.vehicleSoc)

		if se := lp.socEstimator; se != nil {
			if lp.charging() {
				lp.setRemainingDuration(se.RemainingChargeDuration(lp.chargePower, lp.SoC.target))
			} else {
				lp.setRemainingDuration(-1)
			}
		}

		if se := lp.socEstimator; se != nil {
			lp.setRemainingEnergy(1e3 * se.RemainingChargeEnergy(lp.SoC.target))
		}

		// range
		if vs, ok := lp.vehicle.(api.VehicleRange); ok {
			if rng, err := vs.Range(); err == nil {
				lp.log.DEBUG.Printf("vehicle range: %dkm", rng)
				lp.publish(vehicleRange, rng)
			}
		}

		// vehicle target soc
		if vs, ok := lp.vehicle.(api.SocLimiter); ok {
			if targetSoC, err := vs.TargetSoC(); err == nil {
				lp.log.DEBUG.Printf("vehicle target soc: %.0f%%", targetSoC)
				lp.publish(vehicleTargetSoC, targetSoC)
			}
		}

		// trigger message after variables are updated
		lp.bus.Publish(evVehicleSoC, f)
	}
}

// addTask adds a single task to the queue
func (lp *LoadPoint) addTask(task func()) {
	// test guard
	if lp.tasks != nil {
		// don't add twice
		if t, ok := lp.tasks.Peek(); ok &&
			reflect.ValueOf(t).Pointer() == reflect.ValueOf(task).Pointer() {
			return
		}
		lp.tasks.Enqueue(task)
	}
}

// processTasks executes a single task from the queue
func (lp *LoadPoint) processTasks() {
	// test guard
	if lp.tasks != nil {
		if task, ok := lp.tasks.Dequeue(); ok {
			task.(func())()
		}
	}
}

// Update is the main control function. It reevaluates meters and charger state
func (lp *LoadPoint) Update(sitePower float64, cheap, batteryBuffered bool) {
	lp.processTasks()

	mode := lp.GetMode()
	lp.publish("mode", mode)

	// read and publish meters first- charge power has already been updated by the site
	lp.updateChargeCurrents()

	// update ChargeRater here to make sure initial meter update is caught
	lp.bus.Publish(evChargeCurrent, lp.chargeCurrent)
	lp.bus.Publish(evChargePower, lp.chargePower)

	// update progress and soc before status is updated
	lp.publishChargeProgress()

	// read and publish status
	if err := lp.updateChargerStatus(); err != nil {
		lp.log.ERROR.Printf("charger: %v", err)
		return
	}

	lp.publish("connected", lp.connected())
	lp.publish("charging", lp.charging())
	lp.publish("enabled", lp.enabled)

	// identify connected vehicle
	if lp.connected() {
		// read identity and run associated action
		lp.identifyVehicle()

		// find vehicle by status for a couple of minutes after connecting
		if lp.vehicleUnidentified() {
			lp.identifyVehicleByStatus()
		}
	}

	// publish soc after updating charger status to make sure
	// initial update of connected state matches charger status
	lp.publishSoCAndRange()

	// sync settings with charger
	lp.syncCharger()

	// check if car connected and ready for charging
	var err error

	// track if remote disabled is actually active
	remoteDisabled := loadpoint.RemoteEnable

	// reset detection if soc timer needs be deactivated after evaluating the loading strategy
	lp.socTimer.MustValidateDemand()

	// execute loading strategy
	switch {
	case !lp.connected():
		// always disable charger if not connected
		// https://github.com/evcc-io/evcc/issues/105
		err = lp.setLimit(0, false)

	case lp.scalePhasesRequired():
		if err = lp.scalePhases(lp.ConfiguredPhases); err == nil {
			lp.log.DEBUG.Printf("switched phases: %dp", lp.ConfiguredPhases)
		}

	case lp.targetEnergyReached():
		lp.log.DEBUG.Printf("targetEnergy reached: %.0fkWh > %0.1fkWh", lp.getChargedEnergy()/1e3, lp.targetEnergy)
		err = lp.disableUnlessClimater()

	case lp.targetSocReached():
		lp.log.DEBUG.Printf("targetSoC reached: %.1f%% > %d%%", lp.vehicleSoc, lp.SoC.target)
		err = lp.disableUnlessClimater()

	// OCPP has priority over target charging
	case lp.remoteControlled(loadpoint.RemoteHardDisable):
		remoteDisabled = loadpoint.RemoteHardDisable
		fallthrough

	case mode == api.ModeOff:
		err = lp.setLimit(0, true)

	case lp.minSocNotReached():
		// 3p if available
		if err = lp.scalePhasesIfAvailable(3); err == nil {
			err = lp.setLimit(lp.GetMaxCurrent(), true)
		}
		lp.elapsePVTimer() // let PV mode disable immediately afterwards

	case mode == api.ModeNow:
		// 3p if available
		if err = lp.scalePhasesIfAvailable(3); err == nil {
			err = lp.setLimit(lp.GetMaxCurrent(), true)
		}

	// target charging
	case lp.socTimer.DemandActive():
		// 3p if available
		if err = lp.scalePhasesIfAvailable(3); err == nil {
			targetCurrent := lp.socTimer.Handle()
			err = lp.setLimit(targetCurrent, true)
		}

	case mode == api.ModeMinPV || mode == api.ModePV:
		targetCurrent := lp.pvMaxCurrent(mode, sitePower, batteryBuffered)

		var required bool // false
		if targetCurrent == 0 && lp.climateActive() {
			lp.log.DEBUG.Println("climater active")
			targetCurrent = lp.GetMinCurrent()
			required = true
		}

		// tariff
		if cheap {
			targetCurrent = lp.GetMaxCurrent()
			lp.log.DEBUG.Printf("cheap tariff: %.3gA", targetCurrent)
			required = true
		}

		// Sunny Home Manager
		if lp.remoteControlled(loadpoint.RemoteSoftDisable) {
			remoteDisabled = loadpoint.RemoteSoftDisable
			targetCurrent = 0
			required = true
		}

		err = lp.setLimit(targetCurrent, required)
	}

	// Wake-up checks
	if lp.enabled && lp.status == api.StatusB &&
		int(lp.vehicleSoc) < lp.SoC.target && lp.wakeUpTimer.Expired() {
		lp.wakeUpVehicle()
	}

	// stop an active target charging session if not currently evaluated
	if !lp.socTimer.DemandValidated() {
		lp.socTimer.Stop()
	}

	// effective disabled status
	if remoteDisabled != loadpoint.RemoteEnable {
		lp.publish("remoteDisabled", remoteDisabled)
	}

	// log any error
	if err != nil {
		lp.log.ERROR.Println(err)
	}
}
