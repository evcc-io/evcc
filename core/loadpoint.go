package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/db"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"

	evbus "github.com/asaskevich/EventBus"
	"github.com/avast/retry-go/v3"
	"github.com/benbjohnson/clock"
	"github.com/cjrd/allocate"
)

const (
	evChargeStart         = "start"      // update chargeTimer
	evChargeStop          = "stop"       // update chargeTimer
	evChargeCurrent       = "current"    // update fakeChargeMeter
	evChargePower         = "power"      // update chargeRater
	evVehicleConnect      = "connect"    // vehicle connected
	evVehicleDisconnect   = "disconnect" // vehicle disconnected
	evVehicleSoc          = "soc"        // vehicle soc progress
	evVehicleUnidentified = "guest"      // vehicle unidentified

	pvTimer   = "pv"
	pvEnable  = "enable"
	pvDisable = "disable"

	phaseTimer   = "phase"
	phaseScale1p = "scale1p"
	phaseScale3p = "scale3p"

	timerInactive = "inactive"

	minActiveCurrent = 1.0 // minimum current at which a phase is treated as active
	minActiveVoltage = 208 // minimum voltage at which a phase is treated as active

	guardGracePeriod = 10 * time.Second // allow out of sync during this timespan
)

// elapsed is the time an expired timer will be set to
var elapsed = time.Unix(0, 1)

// PollConfig defines the vehicle polling mode and interval
type PollConfig struct {
	Mode     string        `mapstructure:"mode"`     // polling mode charging (default), connected, always
	Interval time.Duration `mapstructure:"interval"` // interval when not charging
}

// SocConfig defines soc settings, estimation and update behaviour
type SocConfig struct {
	Poll     PollConfig `mapstructure:"poll"`
	Estimate *bool      `mapstructure:"estimate"`
	Min_     int        `mapstructure:"min"`    // TODO deprecated
	Target_  int        `mapstructure:"target"` // TODO deprecated
	min      int        // Default minimum Soc, guarded by mutex
	target   int        // Default target Soc, guarded by mutex
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

// Task is the task type
type Task = func()

// Loadpoint is responsible for controlling charge depending on
// Soc needs and power availability.
type Loadpoint struct {
	clock    clock.Clock       // mockable time
	bus      evbus.Bus         // event bus
	pushChan chan<- push.Event // notifications
	uiChan   chan<- util.Param // client push messages
	lpChan   chan<- *Loadpoint // update requests
	log      *util.Logger

	// exposed public configuration
	sync.Mutex                // guard status
	Mode       api.ChargeMode `mapstructure:"mode"` // Charge mode, guarded by mutex

	Title_            string   `mapstructure:"title"`    // UI title
	ConfiguredPhases  int      `mapstructure:"phases"`   // Charger configured phase mode 0/1/3
	ChargerRef        string   `mapstructure:"charger"`  // Charger reference
	VehicleRef        string   `mapstructure:"vehicle"`  // Vehicle reference
	VehiclesRef_      []string `mapstructure:"vehicles"` // TODO deprecated
	MeterRef          string   `mapstructure:"meter"`    // Charge meter reference
	Soc               SocConfig
	Enable, Disable   ThresholdConfig
	ResetOnDisconnect bool `mapstructure:"resetOnDisconnect"`
	onDisconnect      api.ActionConfig
	targetEnergy      float64 // Target charge energy for dumb vehicles in kWh

	MinCurrent    float64       // PV mode: start current	Min+PV mode: min current
	MaxCurrent    float64       // Max allowed current. Physically ensured by the charger
	GuardDuration time.Duration // charger enable/disable minimum holding time

	enabled             bool      // Charger enabled state
	phases              int       // Charger enabled phases, guarded by mutex
	measuredPhases      int       // Charger physically measured phases
	chargeCurrent       float64   // Charger current limit
	guardUpdated        time.Time // Charger enabled/disabled timestamp
	socUpdated          time.Time // Soc updated timestamp (poll: connected)
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

	// target charging
	planner     *planner.Planner
	targetTime  time.Time // time goal
	planSlotEnd time.Time // current plan slot end time
	planActive  bool      // plan is active

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
	vehicleSoc              float64       // Vehicle Soc
	chargeDuration          time.Duration // Charge duration
	chargedEnergy           float64       // Charged energy while connected in Wh
	chargeRemainingDuration time.Duration // Remaining charge duration
	chargeRemainingEnergy   float64       // Remaining charge energy in Wh
	progress                *Progress     // Step-wise progress indicator

	// session log
	db      db.Database
	session *db.Session

	tasks *util.Queue[Task] // tasks to be executed
}

// NewLoadpointFromConfig creates a new loadpoint
func NewLoadpointFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) (*Loadpoint, error) {
	lp := NewLoadpoint(log)
	if err := util.DecodeOther(other, lp); err != nil {
		return nil, err
	}

	// set vehicle polling mode
	switch lp.Soc.Poll.Mode = strings.ToLower(lp.Soc.Poll.Mode); lp.Soc.Poll.Mode {
	case pollCharging:
	case pollConnected, pollAlways:
		lp.log.WARN.Printf("poll mode '%s' may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.Soc.Poll)
	default:
		if lp.Soc.Poll.Mode != "" {
			lp.log.WARN.Printf("invalid poll mode: %s", lp.Soc.Poll.Mode)
		}
		lp.Soc.Poll.Mode = pollCharging
	}

	// set vehicle polling interval
	if lp.Soc.Poll.Interval < pollInterval {
		if lp.Soc.Poll.Interval == 0 {
			lp.Soc.Poll.Interval = pollInterval
		} else {
			lp.log.WARN.Printf("poll interval '%v' is lower than %v and may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.Soc.Poll.Interval, pollInterval)
		}
	}

	if lp.MinCurrent == 0 {
		lp.log.WARN.Println("minCurrent must not be zero")
	}

	if lp.MaxCurrent < lp.MinCurrent {
		lp.log.WARN.Println("maxCurrent must be larger than minCurrent")
	}

	if lp.Soc.Min_ != 0 {
		lp.log.WARN.Println("Configuring soc.min at loadpoint is deprecated and must be applied per vehicle")
	}

	if lp.Soc.Target_ != 0 {
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

// NewLoadpoint creates a Loadpoint with sane defaults
func NewLoadpoint(log *util.Logger) *Loadpoint {
	clock := clock.New()
	bus := evbus.New()

	lp := &Loadpoint{
		log:           log,   // logger
		clock:         clock, // mockable time
		bus:           bus,   // event bus
		Mode:          api.ModeOff,
		status:        api.StatusNone,
		MinCurrent:    6,                                                     // A
		MaxCurrent:    16,                                                    // A
		Soc:           SocConfig{min: 0, target: 100},                        // %
		Enable:        ThresholdConfig{Delay: time.Minute, Threshold: 0},     // t, W
		Disable:       ThresholdConfig{Delay: 3 * time.Minute, Threshold: 0}, // t, W
		GuardDuration: 5 * time.Minute,
		progress:      NewProgress(0, 10),     // soc progress indicator
		coordinator:   coordinator.NewDummy(), // dummy vehicle coordinator
		tasks:         util.NewQueue[Task](),  // task queue
	}

	return lp
}

// collectDefaults collects default values for use on disconnect
func (lp *Loadpoint) collectDefaults() {
	// get reference to action config
	actionCfg := &lp.onDisconnect

	// allocate action config such that all pointer fields are fully allocated
	if err := allocate.Zero(actionCfg); err == nil {
		// initialize with default values
		*actionCfg.Mode = lp.GetMode()
		*actionCfg.MinCurrent = lp.GetMinCurrent()
		*actionCfg.MaxCurrent = lp.GetMaxCurrent()
		*actionCfg.MinSoc = lp.GetMinSoc()
		*actionCfg.TargetSoc = lp.GetTargetSoc()
	} else {
		lp.log.ERROR.Printf("error allocating action config: %v", err)
	}
}

// requestUpdate requests site to update this loadpoint
func (lp *Loadpoint) requestUpdate() {
	select {
	case lp.lpChan <- lp: // request loadpoint update
	default:
	}
}

// configureChargerType ensures that chargeMeter, Rate and Timer can use charger capabilities
func (lp *Loadpoint) configureChargerType(charger api.Charger) {
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
func (lp *Loadpoint) pushEvent(event string) {
	lp.pushChan <- push.Event{Event: event}
}

// publish sends values to UI and databases
func (lp *Loadpoint) publish(key string, val interface{}) {
	if lp.uiChan != nil {
		lp.uiChan <- util.Param{Key: key, Val: val}
	}
}

// evChargeStartHandler sends external start event
func (lp *Loadpoint) evChargeStartHandler() {
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
func (lp *Loadpoint) evChargeStopHandler() {
	lp.log.INFO.Println("stop charging <-")
	lp.pushEvent(evChargeStop)

	// soc update reset
	lp.socUpdated = time.Time{}

	// reset pv enable/disable timer
	// https://github.com/evcc-io/evcc/issues/2289
	if !lp.pvTimer.Equal(elapsed) {
		lp.resetPVTimer()
	}

	lp.stopSession()
}

// evVehicleConnectHandler sends external start event
func (lp *Loadpoint) evVehicleConnectHandler() {
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
func (lp *Loadpoint) evVehicleDisconnectHandler() {
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

	// reset plan once charge goal is met
	lp.setTargetTime(time.Time{})
	lp.setPlanActive(false)
}

// evVehicleSocProgressHandler sends external start event
func (lp *Loadpoint) evVehicleSocProgressHandler(soc float64) {
	if lp.progress.NextStep(soc) {
		lp.pushEvent(evVehicleSoc)
	}
}

// evChargeCurrentHandler publishes the charge current
func (lp *Loadpoint) evChargeCurrentHandler(current float64) {
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
func (lp *Loadpoint) evChargeCurrentWrappedMeterHandler(current float64) {
	power := current * float64(lp.activePhases()) * Voltage

	// if disabled we cannot be charging
	if !lp.enabled || !lp.charging() {
		power = 0
	}

	// handler only called if charge meter was replaced by dummy
	lp.chargeMeter.(*wrapper.ChargeMeter).SetPower(power)
}

// applyAction executes the action
func (lp *Loadpoint) applyAction(actionCfg api.ActionConfig) {
	if actionCfg.Mode != nil {
		lp.SetMode(*actionCfg.Mode)
	}
	if min := actionCfg.MinCurrent; min != nil && *min >= *lp.onDisconnect.MinCurrent {
		lp.SetMinCurrent(*min)
	}
	if max := actionCfg.MaxCurrent; max != nil && *max <= *lp.onDisconnect.MaxCurrent {
		lp.SetMaxCurrent(*max)
	}
	if actionCfg.MinSoc != nil {
		lp.SetMinSoc(*actionCfg.MinSoc)
	}
	if actionCfg.TargetSoc != nil {
		lp.SetTargetSoc(*actionCfg.TargetSoc)
	}
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *Loadpoint) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event, lpChan chan<- *Loadpoint) {
	lp.uiChan = uiChan
	lp.pushChan = pushChan
	lp.lpChan = lpChan

	// event handlers
	_ = lp.bus.Subscribe(evChargeStart, lp.evChargeStartHandler)
	_ = lp.bus.Subscribe(evChargeStop, lp.evChargeStopHandler)
	_ = lp.bus.Subscribe(evVehicleConnect, lp.evVehicleConnectHandler)
	_ = lp.bus.Subscribe(evVehicleDisconnect, lp.evVehicleDisconnectHandler)
	_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentHandler)
	_ = lp.bus.Subscribe(evVehicleSoc, lp.evVehicleSocProgressHandler)

	// publish initial values
	lp.publish(title, lp.Title())
	lp.publish(minCurrent, lp.MinCurrent)
	lp.publish(maxCurrent, lp.MaxCurrent)

	lp.setConfiguredPhases(lp.ConfiguredPhases)
	lp.publish(phasesEnabled, lp.phases)
	lp.publish(phasesActive, lp.activePhases())
	lp.publishTimer(phaseTimer, 0, timerInactive)
	lp.publishTimer(pvTimer, 0, timerInactive)

	// assign and publish default vehicle
	if lp.defaultVehicle != nil {
		lp.setActiveVehicle(lp.defaultVehicle)
	}

	lp.publish("mode", lp.GetMode())
	lp.publish(targetSoc, lp.GetTargetSoc())
	lp.publish(minSoc, lp.GetMinSoc())

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
func (lp *Loadpoint) syncCharger() {
	enabled, err := lp.charger.Enabled()
	if err == nil {
		if enabled != lp.enabled {
			if time.Since(lp.guardUpdated) > guardGracePeriod {
				lp.log.WARN.Printf("charger out of sync: expected %vd, got %vd", status[lp.enabled], status[enabled])
			}
			err = lp.charger.Enable(lp.enabled)
		}

		if !enabled && lp.charging() {
			if time.Since(lp.guardUpdated) > guardGracePeriod {
				lp.log.WARN.Println("charger logic error: disabled but charging")
			}
			err = lp.charger.Enable(false)
		}
	}

	if err != nil {
		lp.log.ERROR.Printf("charger: %v", err)
	}
}

// setLimit applies charger current limits and enables/disables accordingly
func (lp *Loadpoint) setLimit(chargeCurrent float64, force bool) error {
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
func (lp *Loadpoint) connected() bool {
	status := lp.GetStatus()
	return status == api.StatusB || status == api.StatusC
}

// charging returns the EVs charging state
func (lp *Loadpoint) charging() bool {
	return lp.GetStatus() == api.StatusC
}

// charging returns the EVs charging state
func (lp *Loadpoint) setStatus(status api.ChargeStatus) {
	lp.Lock()
	defer lp.Unlock()
	lp.status = status
}

// remainingChargeEnergy returns missing energy amount in kWh if vehicle has a valid energy target
func (lp *Loadpoint) remainingChargeEnergy() (float64, bool) {
	return float64(lp.targetEnergy) - lp.getChargedEnergy()/1e3,
		(lp.vehicle == nil || lp.vehicleHasFeature(api.Offline)) && lp.targetEnergy > 0
}

// targetEnergyReached checks if target is configured and reached
func (lp *Loadpoint) targetEnergyReached() bool {
	f, ok := lp.remainingChargeEnergy()
	return ok && f <= 0
}

// targetSocReached checks if target is configured and reached.
// If vehicle is not configured this will always return false
func (lp *Loadpoint) targetSocReached() bool {
	return lp.vehicle != nil &&
		lp.Soc.target > 0 &&
		lp.Soc.target < 100 &&
		lp.vehicleSoc >= float64(lp.Soc.target)
}

// minSocNotReached checks if minimum is configured and not reached.
// If vehicle is not configured this will always return false
func (lp *Loadpoint) minSocNotReached() bool {
	if lp.vehicle == nil || lp.Soc.min == 0 {
		return false
	}

	if lp.vehicleSoc != 0 {
		return lp.vehicleSoc < float64(lp.Soc.min)
	}

	minEnergy := lp.vehicle.Capacity() * float64(lp.Soc.min) / 100 / soc.ChargeEfficiency
	return minEnergy > 0 && lp.getChargedEnergy() < minEnergy
}

// climateActive checks if vehicle has active climate request
func (lp *Loadpoint) climateActive() bool {
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
func (lp *Loadpoint) disableUnlessClimater() error {
	var current float64 // zero disables
	if lp.climateActive() {
		current = lp.GetMinCurrent()
	}

	// reset plan once charge goal is met
	lp.setPlanActive(false)

	return lp.setLimit(current, true)
}

// remoteControlled returns true if remote control status is active
func (lp *Loadpoint) remoteControlled(demand loadpoint.RemoteDemand) bool {
	lp.Lock()
	defer lp.Unlock()

	return lp.remoteDemand == demand
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
func (lp *Loadpoint) updateChargerStatus() error {
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
func (lp *Loadpoint) effectiveCurrent() float64 {
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
func (lp *Loadpoint) elapsePVTimer() {
	if lp.pvTimer.Equal(elapsed) {
		return
	}

	lp.log.DEBUG.Printf("pv timer elapse")

	lp.pvTimer = elapsed
	lp.guardUpdated = elapsed

	lp.publishTimer(pvTimer, 0, timerInactive)
}

// resetPVTimer resets the pv enable/disable timer to disabled state
func (lp *Loadpoint) resetPVTimer(typ ...string) {
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
func (lp *Loadpoint) resetPhaseTimer() {
	if lp.phaseTimer.IsZero() {
		return
	}

	lp.phaseTimer = time.Time{}
	lp.publishTimer(phaseTimer, 0, timerInactive)
}

// scalePhasesRequired validates if fixed phase configuration matches enabled phases
func (lp *Loadpoint) scalePhasesRequired() bool {
	_, ok := lp.charger.(api.PhaseSwitcher)
	return ok && lp.ConfiguredPhases != 0 && lp.ConfiguredPhases != lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available
func (lp *Loadpoint) scalePhasesIfAvailable(phases int) error {
	if lp.ConfiguredPhases != 0 {
		phases = lp.ConfiguredPhases
	}

	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		return lp.scalePhases(phases)
	}

	return nil
}

// scalePhases adjusts the number of active phases and returns the appropriate charging current.
// Returns api.ErrNotAvailable if api.PhaseSwitcher is not available.
func (lp *Loadpoint) scalePhases(phases int) error {
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

// fastCharging scales to 3p if available and sets maximum current
func (lp *Loadpoint) fastCharging() error {
	err := lp.scalePhasesIfAvailable(3)
	if err == nil {
		err = lp.setLimit(lp.GetMaxCurrent(), true)
	}
	return err
}

// pvScalePhases switches phases if necessary and returns if switch occurred
func (lp *Loadpoint) pvScalePhases(availablePower, minCurrent, maxCurrent float64) bool {
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
func (lp *Loadpoint) coordinatedVehicles() []api.Vehicle {
	if lp.coordinator == nil {
		return nil
	}
	return lp.coordinator.GetVehicles()
}

// TODO move up to timer functions
func (lp *Loadpoint) publishTimer(name string, delay time.Duration, action string) {
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
func (lp *Loadpoint) pvMaxCurrent(mode api.ChargeMode, sitePower float64, batteryBuffered bool) float64 {
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
			lp.resetPVTimer("disable")
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
			lp.resetPVTimer("enable")
		}

		// lp.log.DEBUG.Println("pv enable timer: keep disabled")
		return 0
	}

	// reset timer to disabled state
	lp.resetPVTimer()

	// cap at maximum current
	targetCurrent = math.Min(targetCurrent, maxCurrent)

	return targetCurrent
}

// UpdateChargePower updates charge meter power
func (lp *Loadpoint) UpdateChargePower() {
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

// updateChargeCurrents uses PhaseCurrents interface to count phases with current >=1A
func (lp *Loadpoint) updateChargeCurrents() {
	lp.chargeCurrents = nil

	phaseMeter, ok := lp.chargeMeter.(api.PhaseCurrents)
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
			lp.log.WARN.Printf("invalid phase wiring between charge meter and charger")
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

			lp.log.DEBUG.Printf("detected active phases: %dp", phases)
			lp.publish(phasesActive, phases)
		}
	}
}

// updateChargeVoltages uses PhaseVoltages interface to count phases with nominal grid voltage
func (lp *Loadpoint) updateChargeVoltages() {
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		return // we don't need the voltages
	}

	phaseMeter, ok := lp.chargeMeter.(api.PhaseVoltages)
	if !ok {
		return // don't guess
	}

	u1, u2, u3, err := phaseMeter.Voltages()
	if err != nil {
		lp.log.ERROR.Printf("charge meter: %v", err)
		return
	}

	chargeVoltages := []float64{u1, u2, u3}
	lp.log.DEBUG.Printf("charge voltages: %.3gV", chargeVoltages)
	lp.publish("chargeVoltages", chargeVoltages)

	// Quine-McCluskey for (¬L1∧L2∧¬L3) ∨ (L1∧L2∧¬L3) ∨ (¬L1∧¬L2∧L3) ∨ (L1∧¬L2∧L3) ∨ (¬L1∧L2∧L3) -> ¬L1 ∧ L3 ∨ L2 ∧ ¬L3 ∨ ¬L2 ∧ L3
	if !(u1 > minActiveVoltage) && (u3 > minActiveVoltage) || (u2 > minActiveVoltage) && !(u3 > minActiveVoltage) || !(u2 > minActiveVoltage) && (u3 > minActiveVoltage) {
		lp.log.WARN.Printf("invalid phase wiring between charge meter and charger")
	}

	var phases int
	if (u1 > minActiveVoltage) || (u2 > minActiveVoltage) || (u3 > minActiveVoltage) {
		phases = 3
	}
	if (u1 > minActiveVoltage) && (u2 < minActiveVoltage) && (u3 < minActiveVoltage) {
		phases = 1
	}

	if phases >= 1 {
		lp.log.DEBUG.Printf("detected connected phases: %dp", phases)
		lp.setPhases(phases)
	}
}

// publish charged energy and duration
func (lp *Loadpoint) publishChargeProgress() {
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
func (lp *Loadpoint) socPollAllowed() bool {
	remaining := lp.Soc.Poll.Interval - lp.clock.Since(lp.socUpdated)

	honourUpdateInterval := lp.Soc.Poll.Mode == pollAlways ||
		lp.Soc.Poll.Mode == pollConnected && lp.connected() ||
		lp.Soc.Poll.Mode == pollCharging && lp.connected() && (lp.vehicleSoc < float64(lp.Soc.target))

	if honourUpdateInterval && remaining > 0 {
		lp.log.DEBUG.Printf("next soc poll remaining time: %v", remaining.Truncate(time.Second))
	}

	return lp.charging() || honourUpdateInterval && (remaining <= 0) || lp.connected() && lp.socUpdated.IsZero()
}

// checks if the connected charger can provide Soc to the connected vehicle
func (lp *Loadpoint) socProvidedByCharger() bool {
	if charger, ok := lp.charger.(api.Battery); ok {
		if _, err := charger.Soc(); err == nil {
			return true
		}
	}
	return false
}

// publish state of charge, remaining charge duration and range
func (lp *Loadpoint) publishSocAndRange() {
	// guard for socEstimator removed by api
	if lp.socEstimator == nil {
		return
	}

	if lp.socPollAllowed() || lp.socProvidedByCharger() {
		lp.socUpdated = lp.clock.Now()

		f, err := lp.socEstimator.Soc(lp.getChargedEnergy())
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
		lp.publish(vehicleSoc, lp.vehicleSoc)

		// vehicle target soc
		targetSoc := 100
		if vs, ok := lp.vehicle.(api.SocLimiter); ok {
			if limit, err := vs.TargetSoc(); err == nil {
				targetSoc = int(math.Trunc(limit))
				lp.log.DEBUG.Printf("vehicle soc limit: %.0f%%", limit)
				lp.publish(vehicleTargetSoc, limit)
			} else {
				lp.log.ERROR.Printf("vehicle soc limit: %v", err)
			}
		}

		// use minimum of vehicle and loadpoint
		socLimit := targetSoc
		if lp.Soc.target < socLimit {
			socLimit = lp.Soc.target
		}

		var d time.Duration
		if lp.charging() {
			d = lp.socEstimator.RemainingChargeDuration(socLimit, lp.chargePower)
		}
		lp.SetRemainingDuration(d)

		lp.SetRemainingEnergy(1e3 * lp.socEstimator.RemainingChargeEnergy(socLimit))

		// range
		if vs, ok := lp.vehicle.(api.VehicleRange); ok {
			if rng, err := vs.Range(); err == nil {
				lp.log.DEBUG.Printf("vehicle range: %dkm", rng)
				lp.publish(vehicleRange, rng)
			} else {
				lp.log.ERROR.Printf("vehicle range: %v", err)
			}
		}

		// trigger message after variables are updated
		lp.bus.Publish(evVehicleSoc, f)
	}
}

// addTask adds a single task to the queue
func (lp *Loadpoint) addTask(task func()) {
	// test guard
	if lp.tasks != nil {
		// don't add twice
		if t, ok := lp.tasks.First(); ok &&
			reflect.ValueOf(t).Pointer() == reflect.ValueOf(task).Pointer() {
			return
		}
		lp.tasks.Enqueue(task)
	}
}

// processTasks executes a single task from the queue
func (lp *Loadpoint) processTasks() {
	// test guard
	if lp.tasks != nil {
		if task, ok := lp.tasks.Dequeue(); ok {
			task()
		}
	}
}

// Update is the main control function. It reevaluates meters and charger state
func (lp *Loadpoint) Update(sitePower float64, batteryBuffered bool) {
	lp.processTasks()

	mode := lp.GetMode()
	lp.publish("mode", mode)

	// read and publish meters first- charge power has already been updated by the site
	lp.updateChargeVoltages()
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
	lp.publishSocAndRange()

	// sync settings with charger
	lp.syncCharger()

	// check if car connected and ready for charging
	var err error

	// track if remote disabled is actually active
	remoteDisabled := loadpoint.RemoteEnable

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
		lp.log.DEBUG.Printf("targetSoc reached: %.1f%% > %d%%", lp.vehicleSoc, lp.Soc.target)
		err = lp.disableUnlessClimater()

	// OCPP has priority over target charging
	case lp.remoteControlled(loadpoint.RemoteHardDisable):
		remoteDisabled = loadpoint.RemoteHardDisable
		fallthrough

	case mode == api.ModeOff:
		err = lp.setLimit(0, true)

	// immediate charging
	case mode == api.ModeNow:
		err = lp.fastCharging()

	// minimum or target charging
	case lp.minSocNotReached() || lp.plannerActive():
		err = lp.fastCharging()
		lp.resetPhaseTimer()
		lp.elapsePVTimer() // let PV mode disable immediately afterwards

	case mode == api.ModeMinPV || mode == api.ModePV:
		targetCurrent := lp.pvMaxCurrent(mode, sitePower, batteryBuffered)

		var required bool // false
		if targetCurrent == 0 && lp.climateActive() {
			targetCurrent = lp.GetMinCurrent()
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
		int(lp.vehicleSoc) < lp.Soc.target && lp.wakeUpTimer.Expired() {
		lp.wakeUpVehicle()
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
