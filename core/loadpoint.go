package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/telemetry"
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
	minActiveVoltage = 207 // minimum voltage at which a phase is treated as active

	chargerSwitchDuration = 60 * time.Second // allow out of sync during this timespan
	phaseSwitchDuration   = 60 * time.Second // allow out of sync and do not measure phases during this timespan

	// battery boost states
	boostDisabled = 0
	boostStart    = 1
	boostContinue = 2
)

// elapsed is the time an expired timer will be set to
var elapsed = time.Unix(0, 1)

// Poll modes
const pollInterval = 60 * time.Minute

// Task is the task type
type Task = func()

// Loadpoint is responsible for controlling charge depending on
// Soc needs and power availability.
type Loadpoint struct {
	clock    clock.Clock // mockable time
	bus      evbus.Bus   // event bus
	site     site.API
	pushChan chan<- push.Event // notifications
	uiChan   chan<- util.Param // client push messages
	lpChan   chan<- *Loadpoint // update requests
	log      *util.Logger

	rwMutex      int64        // count reentrant RWMutex
	sync.RWMutex              // guard status
	vmu          sync.RWMutex // guard vehicle

	// exposed public configuration
	CircuitRef string `mapstructure:"circuit"` // Circuit reference
	ChargerRef string `mapstructure:"charger"` // Charger reference
	VehicleRef string `mapstructure:"vehicle"` // Vehicle reference
	MeterRef   string `mapstructure:"meter"`   // Charge meter reference

	Soc             loadpoint.SocConfig
	Enable, Disable loadpoint.ThresholdConfig

	// from yaml
	DefaultMode api.ChargeMode `mapstructure:"mode"`     // Default charge mode, used for disconnect
	Title       string         `mapstructure:"title"`    // UI title
	Priority    int            `mapstructure:"priority"` // Priority

	// from yaml, deprecated
	GuardDuration_ time.Duration `mapstructure:"guardduration"` // ignored, present for compatibility
	Phases_        int           `mapstructure:"phases"`        // ignored, present for compatibility
	MinCurrent_    float64       `mapstructure:"minCurrent"`    // ignored, present for compatibility
	MaxCurrent_    float64       `mapstructure:"maxCurrent"`    // ignored, present for compatibility

	title            string   // UI title
	priority         int      // Priority
	minCurrent       float64  // PV mode: start current	Min+PV mode: min current
	maxCurrent       float64  // Max allowed current. Physically ensured by the charger
	phasesConfigured int      // Charger configured phase mode 0/1/3
	limitSoc         int      // Session limit for soc
	limitEnergy      float64  // Session limit for energy
	smartCostLimit   *float64 // always charge if cost is below this value
	batteryBoost     int      // battery boost state

	mode                api.ChargeMode
	enabled             bool      // Charger enabled state
	phases              int       // Charger enabled phases, guarded by mutex
	measuredPhases      int       // Charger physically measured phases
	offeredCurrent      float64   // Charger current limit
	socUpdated          time.Time // Soc updated timestamp (poll: connected)
	vehicleDetect       time.Time // Vehicle connected timestamp
	chargerSwitched     time.Time // Charger enabled/disabled timestamp
	phasesSwitched      time.Time // Phase switch timestamp
	vehicleDetectTicker *clock.Ticker
	vehicleIdentifier   string

	charger          api.Charger
	chargeTimer      api.ChargeTimer
	chargeRater      api.ChargeRater
	chargedAtStartup float64 // session energy at startup

	circuit        api.Circuit // Circuit
	chargeMeter    api.Meter   // Charger usage meter
	vehicle        api.Vehicle // Currently active vehicle
	defaultVehicle api.Vehicle // Default vehicle (disables detection)
	coordinator    coordinator.API
	socEstimator   *soc.Estimator

	// charge planning
	planner          *planner.Planner
	planTime         time.Time     // time goal
	planPrecondition time.Duration // precondition duration
	planEnergy       float64       // Plan charge energy in kWh (dumb vehicles)
	planSlotEnd      time.Time     // current plan slot end time
	planActive       bool          // charge plan exists and has a currently active slot

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
	energyMetrics           EnergyMetrics // Stats for charged energy by session
	chargeRemainingDuration time.Duration // Remaining charge duration
	chargeRemainingEnergy   float64       // Remaining charge energy in Wh
	progress                *Progress     // Step-wise progress indicator

	// session log
	db      *session.DB
	session *session.Session

	settings settings.Settings

	tasks *util.Queue[Task] // tasks to be executed
}

// NewLoadpointFromConfig creates a new loadpoint
func NewLoadpointFromConfig(log *util.Logger, settings settings.Settings, other map[string]interface{}) (*Loadpoint, error) {
	lp := NewLoadpoint(log, settings)
	if err := util.DecodeOther(other, lp); err != nil {
		return nil, err
	}

	// set vehicle polling mode
	switch lp.Soc.Poll.Mode {
	case loadpoint.PollCharging:
	case loadpoint.PollConnected, loadpoint.PollAlways:
		lp.log.WARN.Printf("poll mode '%s' may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.Soc.Poll)
	default:
		lp.Soc.Poll.Mode = loadpoint.PollCharging
	}

	// validate thresholds
	if lp.Enable.Threshold > lp.Disable.Threshold {
		lp.log.WARN.Printf("PV mode enable threshold (%.0fW) is larger than disable threshold (%.0fW)", lp.Enable.Threshold, lp.Disable.Threshold)
	} else if lp.Enable.Threshold > 0 {
		lp.log.WARN.Printf("PV mode enable threshold %.0fW > 0 will start PV charging on grid power consumption. Did you mean -%.0f?", lp.Enable.Threshold, lp.Enable.Threshold)
	}

	// choose sane default if mode is not set
	if lp.mode = lp.DefaultMode; lp.mode == "" {
		lp.mode = api.ModeOff
	}

	if lp.Title != "" {
		lp.setTitle(lp.Title)
	}

	if lp.Priority > 0 {
		lp.setPriority(lp.Priority)
	}

	if lp.CircuitRef != "" {
		dev, err := config.Circuits().ByName(lp.CircuitRef)
		if err != nil {
			return lp, fmt.Errorf("circuit: %w", err)
		}
		lp.circuit = dev.Instance()
	}

	if lp.MeterRef != "" {
		dev, err := config.Meters().ByName(lp.MeterRef)
		if err != nil {
			return lp, fmt.Errorf("meter: %w", err)
		}
		lp.chargeMeter = dev.Instance()
	}

	// default vehicle
	if lp.VehicleRef != "" {
		dev, err := config.Vehicles().ByName(lp.VehicleRef)
		if err != nil {
			return lp, fmt.Errorf("default vehicle: %w", err)
		}
		lp.defaultVehicle = dev.Instance()
	}

	if lp.ChargerRef == "" {
		return lp, errors.New("missing charger")
	}

	dev, err := config.Chargers().ByName(lp.ChargerRef)
	if err != nil {
		return lp, fmt.Errorf("charger: %w", err)
	}
	lp.charger = dev.Instance()
	lp.configureChargerType(lp.charger)

	// phase switching defaults based on charger capabilities
	if !lp.hasPhaseSwitching() {
		lp.phasesConfigured = 3
		lp.phases = 3
	}

	return lp, nil
}

// NewLoadpoint creates a Loadpoint with sane defaults
func NewLoadpoint(log *util.Logger, settings settings.Settings) *Loadpoint {
	clock := clock.New()
	bus := evbus.New()

	lp := &Loadpoint{
		log:        log,      // logger
		settings:   settings, // settings
		clock:      clock,    // mockable time
		bus:        bus,      // event bus
		mode:       api.ModeOff,
		status:     api.StatusNone,
		minCurrent: 6,  // A
		maxCurrent: 16, // A
		Soc: loadpoint.SocConfig{
			Poll: loadpoint.PollConfig{
				Interval: pollInterval,
				Mode:     loadpoint.PollCharging,
			},
		},
		Enable:      loadpoint.ThresholdConfig{Delay: time.Minute, Threshold: 0},     // t, W
		Disable:     loadpoint.ThresholdConfig{Delay: 3 * time.Minute, Threshold: 0}, // t, W
		progress:    NewProgress(0, 10),                                              // soc progress indicator
		coordinator: coordinator.NewDummy(),                                          // dummy vehicle coordinator
		tasks:       util.NewQueue[Task](),                                           // task queue
	}

	return lp
}

// restoreSettings restores loadpoint settings
func (lp *Loadpoint) restoreSettings() {
	if testing.Testing() {
		return
	}

	// deprecated yaml properties
	if lp.Phases_ > 0 {
		lp.log.WARN.Printf("ignoring deprecated phases: %d. please configure via UI", lp.Phases_)
	}
	if lp.MinCurrent_ > 0 {
		lp.log.WARN.Printf("ignoring deprecated minCurrent: %f. please configure via UI", lp.MinCurrent_)
	}
	if lp.MaxCurrent_ > 0 {
		lp.log.WARN.Printf("ignoring deprecated maxCurrent: %f. please configure via UI", lp.MaxCurrent_)
	}
	if lp.GuardDuration_ > 0 {
		lp.log.WARN.Printf("ignoring deprecated guardduration: %s. please configure via UI", lp.GuardDuration_)
	}

	// restore runtime configuration (database & yaml LPs)
	if v, err := lp.settings.String(keys.Mode); err == nil && v != "" && lp.DefaultMode == api.ModeEmpty {
		lp.setMode(api.ChargeMode(v))
	}
	if v, err := lp.settings.Int(keys.Priority); err == nil {
		lp.setPriority(int(v))
	}
	if v, err := lp.settings.Int(keys.PhasesConfigured); err == nil && (v > 0 || lp.hasPhaseSwitching()) {
		lp.setPhasesConfigured(int(v))
	}
	if v, err := lp.settings.Float(keys.MinCurrent); err == nil && v > 0 {
		lp.setMinCurrent(v)
	}
	if v, err := lp.settings.Float(keys.MaxCurrent); err == nil && v > 0 {
		lp.setMaxCurrent(v)
	}
	if v, err := lp.settings.Int(keys.LimitSoc); err == nil && v > 0 {
		lp.setLimitSoc(int(v))
	}
	if v, err := lp.settings.Float(keys.LimitEnergy); err == nil && v > 0 {
		lp.setLimitEnergy(v)
	}
	if v, err := lp.settings.Float(keys.SmartCostLimit); err == nil {
		lp.SetSmartCostLimit(&v)
	}

	var thresholds loadpoint.ThresholdsConfig
	if err := lp.settings.Json(keys.Thresholds, &thresholds); err == nil {
		lp.setThresholds(thresholds)
	}

	var socConfig loadpoint.SocConfig
	if err := lp.settings.Json(keys.Soc, &socConfig); err == nil {
		lp.setSocConfig(socConfig)
	}

	t, err1 := lp.settings.Time(keys.PlanTime)
	v, err2 := lp.settings.Float(keys.PlanEnergy)
	d, _ := lp.settings.Int(keys.PlanPrecondition)
	if err1 == nil && err2 == nil {
		lp.setPlanEnergy(t, time.Duration(d)*time.Second, v)
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

		// when restarting in the middle of charging session, use this as negative offset
		if f, err := rt.ChargedEnergy(); err == nil {
			lp.chargedAtStartup = f
		}
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
	// test helper
	if lp.uiChan == nil {
		return
	}

	lp.uiChan <- util.Param{Key: key, Val: val}
}

// evChargeStartHandler sends external start event
func (lp *Loadpoint) evChargeStartHandler() {
	lp.log.INFO.Println("start charging ->")
	lp.pushEvent(evChargeStart)

	// charge status
	lp.publish(keys.ChargerStatusReason, api.ReasonUnknown)

	lp.stopWakeUpTimer()

	// soc update reset
	lp.socUpdated = time.Time{}

	// set created when first charging session segment starts
	lp.updateSession(func(session *session.Session) {
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
	util.ResetCached()
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
	lp.energyMetrics.Reset()
	lp.energyMetrics.Publish("session", lp)
	lp.publish(keys.ChargedEnergy, lp.GetChargedEnergy())

	// duration
	lp.connectedTime = lp.clock.Now()
	lp.publish(keys.ConnectedDuration, time.Duration(0))

	// soc update reset
	lp.socUpdated = time.Time{}

	// soc update reset on car change
	if lp.socEstimator != nil {
		lp.socEstimator.Reset()
	}

	// set default or start detection
	if !lp.chargerHasFeature(api.IntegratedDevice) {
		lp.vehicleDefaultOrDetect()
	}

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
	lp.ResetMeasuredPhases()

	// energy and duration
	lp.energyMetrics.Publish("session", lp)
	lp.publish(keys.ChargedEnergy, lp.GetChargedEnergy())
	lp.publish(keys.ConnectedDuration, lp.clock.Since(lp.connectedTime).Round(time.Second))

	// charge status
	lp.publish(keys.ChargerStatusReason, api.ReasonUnknown)

	// forget startup energy offset
	lp.chargedAtStartup = 0

	// remove charger vehicle id and stop potential detection
	lp.setVehicleIdentifier("")
	lp.stopVehicleDetection()

	// set default mode on disconnect
	lp.defaultMode()

	// set default vehicle (may be nil)
	lp.setActiveVehicle(lp.defaultVehicle)

	// soc update reset
	lp.socUpdated = time.Time{}

	// boost
	if err := lp.SetBatteryBoost(false); err != nil {
		lp.log.ERROR.Printf("battery boost: %v", err)
	}

	// reset session
	lp.SetLimitSoc(0)
	lp.SetLimitEnergy(0)

	// mark plan slot as inactive
	// this will force a deletion of an outdated plan once plan time is expired in GetPlan()
	lp.setPlanActive(false)
}

// evVehicleSocProgressHandler sends external start event
func (lp *Loadpoint) evVehicleSocProgressHandler(soc float64) {
	if lp.progress.NextStep(soc) {
		lp.pushEvent(evVehicleSoc)
	}
}

// evChargeCurrentHandler publishes the offered current
func (lp *Loadpoint) evChargeCurrentHandler(current float64) {
	if !lp.enabled {
		current = 0
	}
	lp.publish(keys.OfferedCurrent, current)
}

// evChargeCurrentWrappedMeterHandler updates the dummy charge meter's charge power.
// This simplifies the main flow where the charge meter can always be treated as present.
// It assumes that the charge meter cannot consume more than total household consumption.
// If physical charge meter is present this handler is not used.
// The actual value is published by the evChargeCurrentHandler
func (lp *Loadpoint) evChargeCurrentWrappedMeterHandler(current float64) {
	power := current * float64(lp.ActivePhases()) * Voltage

	// if disabled we cannot be charging
	if !lp.enabled || !lp.charging() {
		power = 0
	}

	// handler only called if charge meter was replaced by dummy
	lp.chargeMeter.(*wrapper.ChargeMeter).SetPower(power)
}

// defaultMode executes the action
func (lp *Loadpoint) defaultMode() {
	lp.RLock()
	mode := lp.DefaultMode
	lp.RUnlock()

	if mode != "" && mode != lp.GetMode() {
		lp.SetMode(mode)
	}
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *Loadpoint) Prepare(site site.API, uiChan chan<- util.Param, pushChan chan<- push.Event, lpChan chan<- *Loadpoint) {
	lp.site = site
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

	// restore settings
	lp.restoreSettings()

	// publish initial values
	lp.publish(keys.Title, lp.GetTitle())
	lp.publish(keys.Mode, lp.GetMode())
	lp.publish(keys.Priority, lp.GetPriority())
	lp.publish(keys.MinCurrent, lp.GetMinCurrent())
	lp.publish(keys.MaxCurrent, lp.GetMaxCurrent())

	lp.publish(keys.EnableThreshold, lp.Enable.Threshold)
	lp.publish(keys.DisableThreshold, lp.Disable.Threshold)
	lp.publish(keys.EnableDelay, lp.Enable.Delay)
	lp.publish(keys.DisableDelay, lp.Disable.Delay)

	if phases := lp.getChargerPhysicalPhases(); phases != 0 {
		if lp.phasesConfigured != phases && lp.phasesConfigured != 0 {
			lp.log.WARN.Printf("configured phases %d do not match physical phases %d", lp.phasesConfigured, phases)
		}
		lp.phases = phases
		lp.phasesConfigured = phases
	}

	lp.publish(keys.PhasesConfigured, lp.phasesConfigured)
	lp.publish(keys.ChargerPhases1p3p, lp.hasPhaseSwitching())
	lp.publish(keys.ChargerSinglePhase, lp.getChargerPhysicalPhases() == 1)
	lp.publish(keys.PhasesActive, lp.ActivePhases())
	lp.publish(keys.SmartCostLimit, lp.smartCostLimit)
	lp.publishTimer(phaseTimer, 0, timerInactive)
	lp.publishTimer(pvTimer, 0, timerInactive)

	// charger features
	for _, f := range []api.Feature{api.IntegratedDevice, api.Heating} {
		lp.publishChargerFeature(f)
	}

	// charger icon
	if c, ok := lp.charger.(api.IconDescriber); ok {
		lp.publish(keys.ChargerIcon, c.Icon())
	} else {
		lp.publish(keys.ChargerIcon, nil)
	}

	// vehicle
	lp.publish(keys.VehicleName, "")
	lp.publish(keys.VehicleOdometer, 0.0)

	// assign and publish default vehicle
	if lp.defaultVehicle != nil {
		lp.setActiveVehicle(lp.defaultVehicle)
	}

	// reset detection state
	lp.publish(keys.VehicleDetectionActive, false)

	// restored settings
	lp.publish(keys.PlanTime, lp.planTime)
	lp.publish(keys.PlanEnergy, lp.planEnergy)
	lp.publish(keys.PlanPrecondition, lp.planPrecondition)
	lp.publish(keys.LimitSoc, lp.limitSoc)
	lp.publish(keys.LimitEnergy, lp.limitEnergy)

	// planner
	lp.publish(keys.PlanActive, lp.planActive)

	// battery boost
	lp.publish(keys.BatteryBoost, lp.batteryBoost != boostDisabled)

	// read initial charger state to prevent immediately disabling charger
	if enabled, err := lp.charger.Enabled(); err == nil {
		if lp.enabled = enabled; enabled {
			// set defined current for use by pv mode
			_ = lp.setLimit(lp.effectiveMinCurrent())
		}
	} else {
		lp.log.ERROR.Printf("charger enabled: %v", err)
	}

	// set vehicle polling mode
	if lp.Soc.Poll.Mode != loadpoint.PollCharging {
		lp.log.WARN.Printf("poll mode '%s' may deplete your battery or lead to API misuse. USE AT YOUR OWN RISK.", lp.Soc.Poll)
	}

	// allow charger to access loadpoint
	if ctrl, ok := lp.charger.(loadpoint.Controller); ok {
		ctrl.LoadpointControl(lp)
	}
}

func (lp *Loadpoint) setAndPublishEnabled(enabled bool) {
	if enabled != lp.enabled {
		lp.log.DEBUG.Printf("charger %s", status[enabled])
		lp.enabled = enabled
	}
	lp.publish(keys.Enabled, enabled)
}

// syncCharger updates charger status and synchronizes it with expectations
func (lp *Loadpoint) syncCharger() error {
	enabled, err := lp.charger.Enabled()
	if err != nil {
		return fmt.Errorf("charger enabled: %w", err)
	}

	shouldBeConsistent := lp.shouldBeConsistent()

	if shouldBeConsistent {
		defer func() {
			lp.setAndPublishEnabled(enabled)
		}()
	}

	// #1: check charger logic, fix charger state if necessary (for chargers that start charging while being disabled)
	if !enabled && lp.charging() {
		lp.log.WARN.Println("charger logic error: disabled but charging")

		// treat as enabled when charging for further validations
		enabled = true

		if shouldBeConsistent {
			if err := lp.charger.Enable(true); err != nil { // also enable charger to correct internal state
				return fmt.Errorf("charger enable: %w", err)
			}

			lp.elapsePVTimer() // elapse PV timer so loadpoint can immediately switch charger if necessary
			return nil
		}
	}

	// #2: sync charger
	switch {
	case enabled && lp.enabled:
		// sync max current
		var (
			current float64
			err     error
		)

		// use chargers actual set current if available
		cg, isCg := lp.charger.(api.CurrentGetter)
		if isCg {
			if current, err = cg.GetMaxCurrent(); err == nil {
				// smallest adjustment most PWM-Controllers can do is: 100%÷256×0,6A = 0.234A
				if delta := math.Abs(lp.offeredCurrent - current); delta > 0.23 {
					if shouldBeConsistent && delta >= 1 {
						lp.log.WARN.Printf("charger logic error: current mismatch (got %.3gA, expected %.3gA) - make sure your interval is at least 30s", current, lp.offeredCurrent)
					}
					lp.offeredCurrent = current
					lp.bus.Publish(evChargeCurrent, lp.offeredCurrent)
				}
			} else if !errors.Is(err, api.ErrNotAvailable) {
				return fmt.Errorf("charger get max current: %w", err)
			}
		}

		// use measured phase currents as fallback if charger does not provide max current or does not currently relay from vehicle (TWC3)
		if !isCg || errors.Is(err, api.ErrNotAvailable) {
			// validate if current too high by more than 1A (https://github.com/evcc-io/evcc/issues/14731)
			if current := lp.GetMaxPhaseCurrent(); current > lp.offeredCurrent+1.0 {
				if shouldBeConsistent {
					lp.log.WARN.Printf("charger logic error: current mismatch (got %.3gA measured, expected %.3gA) - make sure your interval is at least 30s", current, lp.offeredCurrent)
				}
				lp.offeredCurrent = current
				lp.bus.Publish(evChargeCurrent, lp.offeredCurrent)
			}
		}

		// sync phases
		_, isPs := lp.charger.(api.PhaseSwitcher)
		if phases := lp.GetPhases(); isPs && shouldBeConsistent && phases > 0 {
			// fallback to active phases from measured phases
			chargerPhases := lp.measuredPhases
			if chargerPhases == 2 {
				chargerPhases = 3
			}

			pg, isPg := lp.charger.(api.PhaseGetter)
			if isPg {
				if chargerPhases, err = pg.GetPhases(); err == nil {
					if chargerPhases > 0 && chargerPhases != phases {
						lp.log.WARN.Printf("charger logic error: phases mismatch (got %d, expected %d)", chargerPhases, phases)
						lp.SetPhases(chargerPhases)
					}
				} else {
					if errors.Is(err, api.ErrNotAvailable) {
						return nil
					}
					return fmt.Errorf("charger get phases: %w", err)
				}
			}

			// use measured phase currents for active phases as fallback if charger does not provide phases
			if !isPg || errors.Is(err, api.ErrNotAvailable) {
				if chargerPhases > phases {
					lp.log.WARN.Printf("charger logic error: phases mismatch (got %d measured, expected %d)", chargerPhases, phases)
					lp.SetPhases(chargerPhases)
				}
			}
		}

	case enabled == lp.enabled:
		// sync disabled state

	case !enabled && !lp.phaseSwitchCompleted():
		// some chargers (i.E. Easee in some configurations) disable themselves to be able to switch phases
		// -> enable charger
		if err := lp.charger.Enable(true); err != nil {
			return fmt.Errorf("charger enable: %w", err)
		}

	case shouldBeConsistent && (enabled || lp.connected()):
		// ignore disabled state if vehicle was disconnected (!lp.enabled && !lp.connected)
		lp.log.WARN.Printf("charger out of sync: expected %vd, got %vd", status[lp.enabled], status[enabled])
	}

	return nil
}

// coarseCurrent returns true if charger or vehicle require full amp steps
func (lp *Loadpoint) coarseCurrent() bool {
	_, ok := lp.charger.(api.ChargerEx)
	return !ok || lp.vehicleHasFeature(api.CoarseCurrent)
}

// roundedCurrent rounds current down to full amps if charger or vehicle require it
func (lp *Loadpoint) roundedCurrent(current float64) float64 {
	// full amps only?
	if lp.coarseCurrent() {
		current = math.Trunc(current)
	}
	return current
}

// setLimit applies charger current limits and enables/disables accordingly
func (lp *Loadpoint) setLimit(current float64) error {
	current = lp.roundedCurrent(current)

	// apply circuit limits
	if lp.circuit != nil {
		var actualCurrent float64
		if lp.chargeCurrents != nil {
			actualCurrent = max(lp.chargeCurrents[0], lp.chargeCurrents[1], lp.chargeCurrents[2])
		} else if lp.charging() {
			actualCurrent = lp.offeredCurrent
		}

		currentLimit := lp.circuit.ValidateCurrent(actualCurrent, current)

		activePhases := lp.ActivePhases()
		powerLimit := lp.circuit.ValidatePower(lp.chargePower, currentToPower(current, activePhases))
		currentLimitViaPower := powerToCurrent(powerLimit, activePhases)

		current = lp.roundedCurrent(min(currentLimit, currentLimitViaPower))
	}

	// https://github.com/evcc-io/evcc/issues/16309
	effMinCurrent := lp.effectiveMinCurrent()
	if effMaxCurrent := lp.effectiveMaxCurrent(); effMinCurrent > effMaxCurrent {
		return fmt.Errorf("invalid config: min current %.3gA exceeds max current %.3gA", effMinCurrent, effMaxCurrent)
	}

	// set current
	if current != lp.offeredCurrent && current >= effMinCurrent {
		var err error
		if charger, ok := lp.charger.(api.ChargerEx); ok {
			err = charger.MaxCurrentMillis(current)
		} else {
			err = lp.charger.MaxCurrent(int64(current))
		}

		if err != nil {
			v := lp.GetVehicle()
			if vv, ok := v.(api.Resurrector); ok && errors.Is(err, api.ErrAsleep) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				lp.log.DEBUG.Printf("set charge current limit: waking up vehicle")
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("set charge current limit %.3gA: %w", current, err)
		}

		lp.log.DEBUG.Printf("set charge current limit: %.3gA", current)
		lp.offeredCurrent = current
		lp.bus.Publish(evChargeCurrent, current)
	}

	// set enabled/disabled
	if enabled := current >= effMinCurrent; enabled != lp.enabled {
		if err := lp.charger.Enable(enabled); err != nil {
			v := lp.GetVehicle()
			if vv, ok := v.(api.Resurrector); enabled && ok && errors.Is(err, api.ErrAsleep) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				lp.log.DEBUG.Printf("charger %s: waking up vehicle", status[enabled])
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("charger %s: %w", status[enabled], err)
		}

		lp.setAndPublishEnabled(enabled)
		lp.chargerSwitched = lp.clock.Now()

		// ensure we always re-set current when enabling charger
		if !enabled {
			lp.offeredCurrent = 0
		}

		lp.bus.Publish(evChargeCurrent, current)

		// start/stop vehicle wake-up timer
		if enabled {
			lp.startWakeUpTimer()
		} else {
			lp.stopWakeUpTimer()
		}
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

// setStatus updates the internal charging state according to EV
func (lp *Loadpoint) setStatus(status api.ChargeStatus) {
	lp.Lock()
	defer lp.Unlock()
	lp.status = status
}

// socBasedPlanning returns true if vehicle soc (optionally from charger) and capacity are available
func (lp *Loadpoint) socBasedPlanning() bool {
	v := lp.GetVehicle()
	return (v != nil && v.Capacity() > 0) && (lp.vehicleHasSoc() || lp.vehicleSoc > 0)
}

// repeatingPlanning returns true if the current plan is a repeating plan
func (lp *Loadpoint) repeatingPlanning() bool {
	if !lp.socBasedPlanning() {
		return false
	}
	_, _, _, id := lp.NextVehiclePlan()
	return id > 1
}

// vehicleHasSoc returns true if active vehicle supports returning soc, i.e. it is not an offline vehicle
func (lp *Loadpoint) vehicleHasSoc() bool {
	return lp.GetVehicle() != nil && !lp.vehicleHasFeature(api.Offline)
}

// remainingLimitEnergy returns missing energy amount in kWh if vehicle has a valid energy target
func (lp *Loadpoint) remainingLimitEnergy() (float64, bool) {
	limit := lp.getLimitEnergy()
	return max(0, limit-lp.getChargedEnergy()/1e3),
		limit > 0 && !lp.socBasedPlanning()
}

// LimitEnergyReached checks if target is configured and reached
func (lp *Loadpoint) LimitEnergyReached() bool {
	lp.RLock()
	defer lp.RUnlock()
	f, ok := lp.remainingLimitEnergy()
	return ok && f <= 0
}

// LimitSocReached returns true if the effective limit has been reached
func (lp *Loadpoint) LimitSocReached() bool {
	lp.RLock()
	defer lp.RUnlock()
	limit := lp.effectiveLimitSoc()
	return limit > 0 && limit < 100 && lp.vehicleSoc >= float64(limit)
}

// minSocNotReached checks if minimum is configured and not reached.
// If vehicle is not configured this will always return false
func (lp *Loadpoint) minSocNotReached() bool {
	v := lp.GetVehicle()
	if v == nil {
		return false
	}

	minSoc := vehicle.Settings(lp.log, v).GetMinSoc()
	if minSoc == 0 {
		return false
	}

	if lp.vehicleSoc != 0 {
		active := lp.vehicleSoc < float64(minSoc)
		if active {
			lp.log.DEBUG.Printf("forced charging at vehicle soc %.0f%% (< %.0f%% min soc)", lp.vehicleSoc, float64(minSoc))
		}
		return active
	}

	minEnergy := v.Capacity() * float64(minSoc) / 100 / soc.ChargeEfficiency
	return minEnergy > 0 && lp.getChargedEnergy() < minEnergy
}

// disableUnlessClimater disables the charger unless climate is active
func (lp *Loadpoint) disableUnlessClimater() error {
	var current float64 // zero disables
	if lp.vehicleClimateActive() {
		current = lp.effectiveMinCurrent()
	}

	return lp.setLimit(current)
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
func (lp *Loadpoint) updateChargerStatus() (bool, error) {
	var welcomeCharge bool

	status, err := lp.charger.Status()
	if err != nil {
		return false, fmt.Errorf("charger status: %w", err)
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
					welcomeCharge = lp.chargerHasFeature(api.WelcomeCharge)

					// Enable charging on connect if any available vehicle requires it.
					// We're using the PV timer to disable after the welcome
					if !welcomeCharge && !lp.chargerHasFeature(api.IntegratedDevice) {
						for _, v := range lp.availableVehicles() {
							if slices.Contains(v.Features(), api.WelcomeCharge) {
								welcomeCharge = true
								lp.log.DEBUG.Printf("welcome charge: %s", v.Title())
								break
							}
						}
					}

					lp.pushEvent(evVehicleConnect)
				case evVehicleDisconnect:
					lp.pushEvent(evVehicleDisconnect)
				}
			}
		}

		// update whenever there is a state change
		lp.bus.Publish(evChargeCurrent, lp.offeredCurrent)
	}

	return welcomeCharge, nil
}

// effectiveCurrent returns the currently effective charging current
func (lp *Loadpoint) effectiveCurrent() float64 {
	if !lp.charging() {
		return 0
	}

	// adjust actual current for vehicles like Zoe where it remains below target
	if lp.chargeCurrents != nil {
		cur := max(lp.chargeCurrents[0], lp.chargeCurrents[1], lp.chargeCurrents[2])
		return min(cur+2.0, lp.offeredCurrent)
	}

	return lp.offeredCurrent
}

// elapsePVTimer puts the pv enable/disable timer into elapsed state
func (lp *Loadpoint) elapsePVTimer() {
	if lp.pvTimer.Equal(elapsed) {
		return
	}

	lp.log.DEBUG.Printf("pv timer elapse")

	lp.pvTimer = elapsed
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
	lp.log.DEBUG.Println(msg)

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
	return lp.hasPhaseSwitching() && lp.phasesConfigured != 0 && lp.phasesConfigured != lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available
func (lp *Loadpoint) scalePhasesIfAvailable(phases int) error {
	if lp.phasesConfigured != 0 {
		phases = lp.phasesConfigured
	}

	if lp.hasPhaseSwitching() {
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
		// switch phases
		if err := cp.Phases1p3p(phases); err != nil {
			return fmt.Errorf("switch phases: %w", err)
		}

		lp.log.DEBUG.Printf("switched phases: %dp", phases)

		// prevent premature measurement of active phases
		lp.phasesSwitched = lp.clock.Now()

		// update setting and reset timer
		lp.SetPhases(phases)
	}

	return nil
}

// fastCharging scales to 3p if available and sets maximum current
func (lp *Loadpoint) fastCharging() error {
	err := lp.scalePhasesIfAvailable(3)
	if err == nil {
		err = lp.setLimit(lp.effectiveMaxCurrent())
	}
	return err
}

// pvScalePhases switches phases if necessary and returns number of phases switched to
func (lp *Loadpoint) pvScalePhases(sitePower, minCurrent, maxCurrent float64) int {
	phases := lp.GetPhases()

	// observed phase state inconsistency
	// - https://github.com/evcc-io/evcc/issues/1572
	// - https://github.com/evcc-io/evcc/issues/2230
	// - https://github.com/evcc-io/evcc/issues/2613
	measuredPhases := lp.GetMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		if lp.chargerUpdateCompleted() && lp.phaseSwitchCompleted() {
			lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
		}
		lp.ResetMeasuredPhases()
	}

	var waiting bool
	activePhases := lp.ActivePhases()
	availablePower := lp.chargePower - sitePower
	scalable := (sitePower > 0 || !lp.enabled) && activePhases > 1 && lp.phasesConfigured < 3

	// scale down phases
	if targetCurrent := powerToCurrent(availablePower, activePhases); targetCurrent < minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)

		if !lp.charging() { // scale immediately if not charging
			lp.phaseTimer = elapsed
		}

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.GetDisableDelay(), phaseScale1p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.GetDisableDelay() {
			if err := lp.scalePhases(1); err != nil {
				lp.log.ERROR.Println(err)
			}
			return 1
		}

		waiting = true
	}

	maxPhases := lp.MaxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)
	scalable = maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, 3*Voltage*minCurrent, maxPhases)

		if !lp.charging() { // scale immediately if not charging
			lp.phaseTimer = elapsed
		}

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.GetEnableDelay(), phaseScale3p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.GetEnableDelay() {
			if err := lp.scalePhases(3); err != nil {
				lp.log.ERROR.Println(err)
			}
			return 3
		}

		waiting = true
	}

	// reset timer to disabled state
	if !waiting && !lp.phaseTimer.IsZero() {
		lp.resetPhaseTimer()
	}

	return 0
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

// boostPower returns the additional power that the loadpoint should draw from the battery
func (lp *Loadpoint) boostPower(batteryBoostPower float64) float64 {
	boost := lp.GetBatteryBoost()
	if boost == boostDisabled {
		return 0
	}

	// push demand to drain battery (at least 100W)
	delta := math.Max(100, math.Abs(lp.site.GetResidualPower()))

	if lp.coarseCurrent() {
		// add effective step power to delta to make sure to step up to the next full amp
		// just using lp.EffectiveStepPower() as delta is not enough because this will result
		// in a too low current when there is a bit remaining grid consumption due to the accuracy
		// of the battery controller
		delta += lp.EffectiveStepPower()
	}

	// start boosting by setting maximum power
	if boost == boostStart {
		delta = lp.EffectiveMaxPower()

		// expire timers
		lp.phaseTimer = elapsed
		lp.pvTimer = elapsed

		if lp.charging() {
			lp.setBatteryBoost(boostContinue)
		}
	}

	res := batteryBoostPower + delta + lp.site.GetResidualPower()
	lp.log.DEBUG.Printf("pv charge battery boost: %.0fW = -%.0fW battery - %.0fW boost", -res, batteryBoostPower, delta)

	return res
}

// pvMaxCurrent calculates the maximum target current for PV mode
func (lp *Loadpoint) pvMaxCurrent(mode api.ChargeMode, sitePower, batteryBoostPower float64, batteryBuffered, batteryStart bool) float64 {
	// read only once to simplify testing
	minCurrent := lp.effectiveMinCurrent()
	maxCurrent := lp.effectiveMaxCurrent()

	// push demand to drain battery
	sitePower -= lp.boostPower(batteryBoostPower)

	// switch phases up/down
	var scaledTo int
	if lp.hasPhaseSwitching() && lp.phaseSwitchCompleted() {
		scaledTo = lp.pvScalePhases(sitePower, minCurrent, maxCurrent)
	}

	// calculate target charge current from delta power and actual current
	activePhases := lp.ActivePhases()
	effectiveCurrent := lp.effectiveCurrent()
	if scaledTo == 3 {
		// if we did scale, adjust the effective current to the new phase count
		effectiveCurrent /= float64(lp.maxActivePhases())
	}
	if lp.chargerHasFeature(api.IntegratedDevice) {
		// for slow-acting heating devices, only take actually consumed power into account
		effectiveCurrent = powerToCurrent(lp.chargePower, activePhases)
	}
	deltaCurrent := powerToCurrent(-sitePower, activePhases)
	targetCurrent := max(effectiveCurrent+deltaCurrent, 0)

	// in MinPV mode or under special conditions return at least minCurrent
	if battery := batteryStart || batteryBuffered && lp.charging(); (mode == api.ModeMinPV || battery) && targetCurrent < minCurrent {
		lp.log.DEBUG.Printf("pv charge current: min %.3gA > %.3gA (%.0fW @ %dp, battery: %t)", minCurrent, targetCurrent, sitePower, activePhases, battery)
		return minCurrent
	}

	lp.log.DEBUG.Printf("pv charge current: %.3gA = %.3gA + %.3gA (%.0fW @ %dp)", targetCurrent, effectiveCurrent, deltaCurrent, sitePower, activePhases)

	if mode == api.ModePV && lp.enabled && targetCurrent < minCurrent {
		projectedSitePower := sitePower
		if !lp.phaseTimer.IsZero() {
			// calculate site power after a phase switch from activePhases phases -> 1 phase
			// notes: activePhases can be 1, 2 or 3 and phaseTimer can only be active if lp current is already at minCurrent
			projectedSitePower -= Voltage * minCurrent * float64(activePhases-1)
		}
		// kick off disable sequence
		if projectedSitePower >= lp.Disable.Threshold {
			lp.log.DEBUG.Printf("projected site power %.0fW >= %.0fW disable threshold", projectedSitePower, lp.Disable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv disable timer start: %v", lp.GetDisableDelay())
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.GetDisableDelay(), pvDisable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.GetDisableDelay() {
				lp.log.DEBUG.Println("pv disable timer elapsed")

				// reset timer to prevent immediate charger re-enabling
				lp.resetPVTimer()

				return 0
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv disable timer remaining: %v", (lp.GetDisableDelay() - elapsed).Round(time.Second))
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
				lp.log.DEBUG.Printf("pv enable timer start: %v", lp.GetEnableDelay())
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.GetEnableDelay(), pvEnable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.GetEnableDelay() {
				lp.log.DEBUG.Println("pv enable timer elapsed")

				// reset timer to prevent immediate charger re-disabling
				lp.resetPVTimer()

				return minCurrent
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv enable timer remaining: %v", (lp.GetEnableDelay() - elapsed).Round(time.Second))
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
	targetCurrent = min(targetCurrent, maxCurrent)

	return targetCurrent
}

// UpdateChargePowerAndCurrents updates charge meter power and currents for load management
func (lp *Loadpoint) UpdateChargePowerAndCurrents() float64 {
	power, err := backoff.RetryWithData(lp.chargeMeter.CurrentPower, modbus.Backoff())
	if err == nil {
		lp.Lock()
		lp.chargePower = power // update value if no error
		lp.Unlock()

		lp.log.DEBUG.Printf("charge power: %.0fW", power)
		lp.publish(keys.ChargePower, power)

		// https://github.com/evcc-io/evcc/issues/2153
		// https://github.com/evcc-io/evcc/issues/6986
		// https://github.com/evcc-io/evcc/issues/13378
		if power < -100 && lp.shouldBeConsistent() {
			lp.log.WARN.Printf("charge power must not be negative: %.0f", power)
		}
	} else {
		power = 0
		lp.log.ERROR.Printf("charge power: %v", err)
	}

	// update charge currents
	lp.chargeCurrents = nil

	if phaseMeter, ok := lp.chargeMeter.(api.PhaseCurrents); ok {
		if err := backoff.Retry(func() error {
			i1, i2, i3, err := phaseMeter.Currents()
			if err != nil {
				if errors.Is(err, api.ErrNotAvailable) {
					err = backoff.Permanent(err)
				}
				return err
			}

			lp.Lock()
			lp.chargeCurrents = []float64{i1, i2, i3}
			lp.Unlock()

			lp.log.DEBUG.Printf("charge currents: %.3gA", lp.chargeCurrents)
			lp.publish(keys.ChargeCurrents, lp.chargeCurrents)

			return nil
		}, modbus.Backoff()); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("charge currents: %v", err)
		}
	}

	return power
}

// phasesFromChargeCurrents uses PhaseCurrents interface to count phases with current >=1A
func (lp *Loadpoint) phasesFromChargeCurrents() {
	if lp.chargeCurrents == nil {
		return
	}

	if lp.charging() && lp.phaseSwitchCompleted() {
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
			lp.publish(keys.PhasesActive, phases)
		}
	}
}

// updateChargeVoltages uses PhaseVoltages interface to count phases with nominal grid voltage
func (lp *Loadpoint) updateChargeVoltages() {
	phaseMeter, ok := lp.chargeMeter.(api.PhaseVoltages)
	if !ok {
		return // don't guess
	}

	u1, u2, u3, err := phaseMeter.Voltages()
	if err != nil {
		// phaseSwitching devices may announce voltages but doesn't deliver
		if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("charge voltages: %v", err)
		}
		return
	}

	chargeVoltages := []float64{u1, u2, u3}
	lp.log.DEBUG.Printf("charge voltages: %.3gV", chargeVoltages)
	lp.publish(keys.ChargeVoltages, chargeVoltages)

	if lp.hasPhaseSwitching() {
		return // we don't need the voltages, but publish
	}

	a1, a2, a3 := u1 >= minActiveVoltage, u2 >= minActiveVoltage, u3 >= minActiveVoltage

	// Quine-McCluskey for (¬L1∧L2∧¬L3) ∨ (L1∧L2∧¬L3) ∨ (¬L1∧¬L2∧L3) ∨ (L1∧¬L2∧L3) ∨ (¬L1∧L2∧L3) -> ¬L1 ∧ L3 ∨ L2 ∧ ¬L3 ∨ ¬L2 ∧ L3
	if !a1 && a3 || a2 && !a3 || !a2 && a3 {
		lp.log.WARN.Printf("invalid phase wiring between charge meter and charger")
	}

	var phases int
	if a1 || a2 || a3 {
		phases = 3
	}
	if a1 && !a2 && !a3 {
		phases = 1
	}

	if phases >= 1 {
		lp.log.DEBUG.Printf("detected connected phases: %dp", phases)
		lp.SetPhases(phases)
	}
}

// publish charged energy and duration
func (lp *Loadpoint) publishChargeProgress() {
	if f, err := lp.chargeRater.ChargedEnergy(); err == nil {
		// workaround for Go-E resetting during disconnect, see
		// https://github.com/evcc-io/evcc/issues/5092
		if f > lp.chargedAtStartup {
			added, addedGreen := lp.energyMetrics.Update(f - lp.chargedAtStartup)
			if telemetry.Enabled() && added > 0 {
				telemetry.UpdateEnergy(added, addedGreen)
			}
		}
	} else {
		lp.log.ERROR.Printf("charge rater: %v", err)
	}

	if d, err := lp.chargeTimer.ChargeDuration(); err == nil {
		lp.chargeDuration = d.Round(time.Second)
	} else {
		lp.log.ERROR.Printf("charge timer: %v", err)
	}

	// TODO check if "session" prefix required?
	lp.energyMetrics.Publish("session", lp)

	// TODO deprecated: use sessionEnergy instead
	lp.publish(keys.ChargedEnergy, lp.GetChargedEnergy())
	lp.publish(keys.ChargeDuration, lp.chargeDuration)
	if _, ok := lp.chargeMeter.(api.MeterEnergy); ok {
		lp.publish(keys.ChargeTotalImport, lp.chargeMeterTotal())
	}
}

// publish state of charge, remaining charge duration and range
//
// - online vehicle connected: this allows estimating remaining energy/duration
//   - either charger or vehicle provides soc
//   - estimator is responsible for querying both
//
// - offline or no vehicle connected (e.g. integrated device): missing capacity, hence no estimate
//   - charger may still provide soc
//   - no estimator
func (lp *Loadpoint) publishSocAndRange() {
	// guard for socEstimator removed by api and keep a local copy in order to avoid race conditions
	// https://github.com/evcc-io/evcc/issues/16180
	socEstimator := lp.socEstimator

	// capacity not available
	if socEstimator == nil || !lp.vehicleHasSoc() {
		if soc, err := lp.chargerSoc(); err == nil {
			lp.vehicleSoc = soc
			lp.publish(keys.VehicleSoc, lp.vehicleSoc)

			if vs, ok := lp.charger.(api.SocLimiter); ok {
				if limit, err := vs.GetLimitSoc(); err == nil {
					lp.log.DEBUG.Printf("charger soc limit: %d%%", limit)
					// https://github.com/evcc-io/evcc/issues/13349
					lp.publish(keys.VehicleLimitSoc, float64(limit))
				} else if !errors.Is(err, api.ErrNotAvailable) {
					lp.log.ERROR.Printf("charger soc limit: %v", err)
				}
			}
		} else if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("charger soc: %v", err)
		}

		return
	}

	// integrated device can bypass the update interval if vehicle is separately configured (legacy)
	if lp.chargerHasFeature(api.IntegratedDevice) || lp.vehicleSocPollAllowed() {
		lp.socUpdated = lp.clock.Now()

		f, err := socEstimator.Soc(lp.GetChargedEnergy())
		if err != nil {
			if loadpoint.AcceptableError(err) {
				lp.socUpdated = time.Time{}
			} else {
				lp.log.ERROR.Printf("vehicle soc: %v", err)
			}

			return
		}

		lp.vehicleSoc = f
		lp.log.DEBUG.Printf("vehicle soc: %.0f%%", lp.vehicleSoc)
		lp.publish(keys.VehicleSoc, lp.vehicleSoc)

		// vehicle target soc
		// TODO take vehicle api limits into account
		apiLimitSoc := 100

		// vehicle limit
		if vs, ok := lp.GetVehicle().(api.SocLimiter); ok {
			if limit, err := vs.GetLimitSoc(); err == nil {
				apiLimitSoc = int(limit)
				lp.log.DEBUG.Printf("vehicle soc limit: %d%%", limit)
				// https://github.com/evcc-io/evcc/issues/13349
				lp.publish(keys.VehicleLimitSoc, float64(limit))
			} else if !loadpoint.AcceptableError(err) {
				lp.log.ERROR.Printf("vehicle soc limit: %v", err)
			}
		}

		// use minimum of vehicle and loadpoint
		limitSoc := min(apiLimitSoc, lp.EffectiveLimitSoc())

		var d time.Duration
		if lp.charging() {
			d = socEstimator.RemainingChargeDuration(limitSoc, lp.chargePower)
		}
		lp.SetRemainingDuration(d)

		lp.SetRemainingEnergy(1e3 * socEstimator.RemainingChargeEnergy(limitSoc))

		// range
		if vs, ok := lp.GetVehicle().(api.VehicleRange); ok {
			if rng, err := vs.Range(); err == nil {
				lp.log.DEBUG.Printf("vehicle range: %dkm", rng)
				lp.publish(keys.VehicleRange, rng)
			} else if !loadpoint.AcceptableError(err) {
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

// startWakeUpTimer starts wakeUpTimer
func (lp *Loadpoint) startWakeUpTimer() {
	lp.log.DEBUG.Printf("wake-up timer: start")
	lp.wakeUpTimer.Start()
}

// stopWakeUpTimer stops wakeUpTimer
func (lp *Loadpoint) stopWakeUpTimer() {
	lp.log.DEBUG.Printf("wake-up timer: stop")
	lp.wakeUpTimer.Stop()
}

func (lp *Loadpoint) shouldBeConsistent() bool {
	return lp.chargerUpdateCompleted() && lp.phaseSwitchCompleted()
}

// chargerUpdateCompleted returns true if enable command should be already processed by the charger (so we can try to sync charger and loadpoint)
func (lp *Loadpoint) chargerUpdateCompleted() bool {
	return time.Since(lp.chargerSwitched) > chargerSwitchDuration
}

// phaseSwitchCompleted returns true if phase switch command should be already processed by the charger (so we can try to sync charger and loadpoint and are able to measure currents)
func (lp *Loadpoint) phaseSwitchCompleted() bool {
	return time.Since(lp.phasesSwitched) > phaseSwitchDuration
}

// Update is the main control function. It reevaluates meters and charger state
func (lp *Loadpoint) Update(sitePower, batteryBoostPower float64, rates api.Rates, batteryBuffered, batteryStart bool, greenShare float64, effPrice, effCo2 *float64) {
	// smart cost
	smartCostActive := lp.smartCostActive(rates)
	lp.publish(keys.SmartCostActive, smartCostActive)

	var smartCostNextStart time.Time
	if !smartCostActive {
		smartCostNextStart = lp.smartCostNextStart(rates)
	}
	lp.publish(keys.SmartCostNextStart, smartCostNextStart)

	// long-running tasks
	lp.processTasks()

	// read and publish meters first- charge power and currents have already been updated by the site
	lp.updateChargeVoltages()
	lp.phasesFromChargeCurrents()

	lp.energyMetrics.SetEnvironment(greenShare, effPrice, effCo2)

	// update ChargeRater here to make sure initial meter update is caught
	lp.bus.Publish(evChargeCurrent, lp.offeredCurrent)
	lp.bus.Publish(evChargePower, lp.chargePower)

	// update progress and soc before status is updated
	lp.publishChargeProgress()
	lp.PublishEffectiveValues()

	// read and publish status
	welcomeCharge, err := lp.updateChargerStatus()
	if err != nil {
		lp.log.ERROR.Println(err)
		return
	}

	lp.publish(keys.VehicleWelcomeActive, welcomeCharge)
	lp.publish(keys.Connected, lp.connected())
	lp.publish(keys.Charging, lp.charging())

	if sr, ok := lp.charger.(api.StatusReasoner); ok && lp.GetStatus() == api.StatusB {
		if r, err := sr.StatusReason(); err == nil {
			lp.publish(keys.ChargerStatusReason, r)
		} else {
			lp.log.ERROR.Printf("charger status reason: %v", err)
		}
	}

	// identify connected vehicle
	if lp.connected() && !lp.chargerHasFeature(api.IntegratedDevice) {
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
	if err := lp.syncCharger(); err != nil {
		lp.log.ERROR.Println(err)
		return
	}

	// track if remote disabled is actually active
	remoteDisabled := loadpoint.RemoteEnable

	mode := lp.GetMode()
	lp.publish(keys.Mode, mode)

	// update and publish plan without being short-circuited by modes etc.
	plannerActive := lp.plannerActive()

	// execute loading strategy
	switch {
	case !lp.connected():
		// always disable charger if not connected
		// https://github.com/evcc-io/evcc/issues/105
		err = lp.setLimit(0)

	case lp.scalePhasesRequired():
		err = lp.scalePhases(lp.phasesConfigured)

	case lp.remoteControlled(loadpoint.RemoteHardDisable):
		remoteDisabled = loadpoint.RemoteHardDisable
		fallthrough

	case mode == api.ModeOff:
		var current float64
		if welcomeCharge {
			current = lp.effectiveMinCurrent()
		}
		err = lp.setLimit(current)

	// minimum or target charging
	case lp.minSocNotReached() || plannerActive:
		err = lp.fastCharging()
		lp.resetPhaseTimer()
		lp.elapsePVTimer() // let PV mode disable immediately afterwards

	case lp.LimitEnergyReached():
		lp.log.DEBUG.Printf("limitEnergy reached: %.0fkWh > %0.1fkWh", lp.GetChargedEnergy()/1e3, lp.limitEnergy)
		err = lp.disableUnlessClimater()

	case lp.LimitSocReached():
		lp.log.DEBUG.Printf("limitSoc reached: %.1f%% > %d%%", lp.vehicleSoc, lp.EffectiveLimitSoc())
		err = lp.disableUnlessClimater()

	// immediate charging- must be placed after limits are evaluated
	case mode == api.ModeNow:
		err = lp.fastCharging()

	case mode == api.ModeMinPV || mode == api.ModePV:
		// cheap tariff
		if smartCostActive {
			rate, _ := rates.At(time.Now())
			lp.log.DEBUG.Printf("smart cost active: %.2f", rate.Value)
			err = lp.fastCharging()
			lp.resetPhaseTimer()
			lp.elapsePVTimer() // let PV mode disable immediately afterwards
			break
		}

		targetCurrent := lp.pvMaxCurrent(mode, sitePower, batteryBoostPower, batteryBuffered, batteryStart)

		if targetCurrent == 0 && lp.vehicleClimateActive() {
			targetCurrent = lp.effectiveMinCurrent()
		}

		if targetCurrent == 0 && welcomeCharge {
			targetCurrent = lp.effectiveMinCurrent()
			lp.resetPVTimer()
		}

		// Sunny Home Manager
		if lp.remoteControlled(loadpoint.RemoteSoftDisable) {
			remoteDisabled = loadpoint.RemoteSoftDisable
			targetCurrent = 0
		}

		err = lp.setLimit(targetCurrent)
	}

	// Wake-up checks
	if lp.enabled && lp.status == api.StatusB &&
		// TODO take vehicle api limits into account
		int(lp.vehicleSoc) < lp.EffectiveLimitSoc() && lp.wakeUpTimer.Expired() {
		lp.wakeUpVehicle()
	}

	// effective disabled status
	if remoteDisabled != loadpoint.RemoteEnable {
		lp.publish(keys.RemoteDisabled, remoteDisabled)
	}

	// log any error
	if err != nil {
		lp.log.ERROR.Println(err)
	}
}
