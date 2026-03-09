package chargercontroller

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	Voltage = 230 // V

	minActiveCurrent = 1.0 // minimum current at which a phase is treated as active

	chargerSwitchDuration = 60 * time.Second
	phaseSwitchDuration   = 60 * time.Second

	phaseTimer   = "phase"
	phaseScale1p = "scale1p"
	phaseScale3p = "scale3p"

	timerInactive = "inactive"
)

// unknownPhases is the assumed phase count for switchable chargers during startup
const unknownPhases = 3

// elapsed is the time an expired timer will be set to
var elapsed = time.Unix(0, 1)

var enabledStatus = map[bool]string{false: "disable", true: "enable"}

// CurrentController controls current-based chargers (EV wallboxes).
// It converts power targets to current, manages phase switching,
// and handles charger enable/disable with vehicle wake-up.
type CurrentController struct {
	log   *util.Logger
	clock clock.Clock
	host  Host // loadpoint callbacks

	// charger interfaces
	charger        api.Charger
	chargerEx      api.ChargerEx      // optional: milliamp precision
	phaseSwitcher  api.PhaseSwitcher  // optional: 1p/3p switching
	phaseDescriber api.PhaseDescriber // optional: physical phase count
	currentLimiter api.CurrentLimiter // optional: charger min/max current
	circuit        api.Circuit        // optional: load management

	// configuration
	minCurrent       float64
	maxCurrent       float64
	phasesConfigured int // 0=auto, 1, 3

	// charger control state
	offeredCurrent  float64
	enabled         bool
	chargerSwitched time.Time

	// phase state
	phases         int       // charger enabled phases
	measuredPhases int       // physically measured phases
	phaseTimer     time.Time // 1p3p switch timer
	phasesSwitched time.Time // last phase switch timestamp

	// measurement state
	chargePower    float64   // measured charge power
	chargeCurrents []float64 // measured per-phase currents

	// timing configuration
	enableDelay  time.Duration
	disableDelay time.Duration

	// callbacks
	setEnabled func(bool)
	publish    func(key string, val any)
}

// NewCurrentController creates a CurrentController for a standard current-based charger.
func NewCurrentController(
	log *util.Logger,
	clck clock.Clock,
	host Host,
	charger api.Charger,
	circuit api.Circuit,
	minCurrent, maxCurrent float64,
	phasesConfigured int,
	phases int,
	enableDelay, disableDelay time.Duration,
	setEnabled func(bool),
	publish func(string, any),
) *CurrentController {
	c := &CurrentController{
		log:              log,
		clock:            clck,
		host:             host,
		charger:          charger,
		circuit:          circuit,
		minCurrent:       minCurrent,
		maxCurrent:       maxCurrent,
		phasesConfigured: phasesConfigured,
		phases:           phases,
		enableDelay:      enableDelay,
		disableDelay:     disableDelay,
		setEnabled:       setEnabled,
		publish:          publish,
	}

	// detect optional charger capabilities
	if ex, ok := charger.(api.ChargerEx); ok {
		c.chargerEx = ex
	}
	if ps, ok := charger.(api.PhaseSwitcher); ok {
		c.phaseSwitcher = ps
	}
	if pd, ok := charger.(api.PhaseDescriber); ok {
		c.phaseDescriber = pd
	}
	if cl, ok := charger.(api.CurrentLimiter); ok {
		c.currentLimiter = cl
	}

	return c
}

// SyncState synchronizes the controller's enabled and offered current state
// with the initial charger state discovered during Prepare().
func (c *CurrentController) SyncState(enabled bool) {
	c.enabled = enabled
	if enabled {
		c.offeredCurrent = c.minCurrent
	}
}

// --- Controller interface ---

func (c *CurrentController) SetOfferedPower(power float64) error {
	// 1. Phase optimization
	if c.hasPhaseSwitching() && c.phaseSwitchCompleted() {
		c.optimizePhases(power)
	}

	// 2. Convert power to current
	activePhases := c.activePhases()
	current := powerToCurrent(power, activePhases)
	current = c.roundedCurrent(current)

	// 3. Apply circuit limits
	if c.circuit != nil {
		var actualCurrent float64
		if c.chargeCurrents != nil {
			actualCurrent = max(c.chargeCurrents[0], c.chargeCurrents[1], c.chargeCurrents[2])
		} else if c.host.Charging() {
			actualCurrent = c.offeredCurrent
		}

		currentLimit := c.circuit.ValidateCurrent(actualCurrent, current)
		powerLimit := c.circuit.ValidatePower(c.chargePower, currentToPower(current, activePhases))
		currentLimitViaPower := powerToCurrent(powerLimit, activePhases)
		current = c.roundedCurrent(min(currentLimit, currentLimitViaPower))
	}

	// 4. Validate min/max
	effMinCurrent := c.effectiveMinCurrent()
	if effMaxCurrent := c.effectiveMaxCurrent(); effMinCurrent > effMaxCurrent {
		return fmt.Errorf("invalid config: min current %.3gA exceeds max current %.3gA", effMinCurrent, effMaxCurrent)
	}

	// 5. Set current on charger
	if current != c.offeredCurrent && current >= effMinCurrent {
		if err := c.setChargerCurrent(current); err != nil {
			return err
		}
	}

	// 6. Enable/disable
	if enabled := current >= effMinCurrent; enabled != c.enabled {
		if err := c.charger.Enable(enabled); err != nil {
			if enabled && errors.Is(err, api.ErrAsleep) {
				c.log.DEBUG.Printf("charger %s: waking up vehicle", enabledStatus[enabled])
				if wakeErr := c.host.WakeUpVehicle(); wakeErr != nil {
					return fmt.Errorf("wake-up vehicle: %w", wakeErr)
				}
			}
			return fmt.Errorf("charger %s: %w", enabledStatus[enabled], err)
		}

		c.enabled = enabled
		c.setEnabled(enabled)
		c.chargerSwitched = c.clock.Now()

		if !enabled {
			c.offeredCurrent = 0
		}

		if enabled {
			c.host.StartWakeUpTimer()
		} else {
			c.host.StopWakeUpTimer()
		}
	}

	return nil
}

func (c *CurrentController) SetMaxPower() error {
	if c.hasPhaseSwitching() {
		phases := 3

		// load management limit active
		if c.circuit != nil {
			minPower3p := currentToPower(c.effectiveMinCurrent(), 3)
			if powerLimit := c.circuit.ValidatePower(c.chargePower, minPower3p); powerLimit < minPower3p {
				phases = 1
				c.log.DEBUG.Printf("fast charging: scaled to 1p to match %.0fW available circuit power", powerLimit)
			}
		}

		if err := c.scalePhasesIfAvailable(phases); err != nil {
			return err
		}
	}

	return c.SetOfferedPower(currentToPower(c.effectiveMaxCurrent(), c.activePhases()))
}

func (c *CurrentController) MinPower() float64 {
	return Voltage * c.effectiveMinCurrent() * float64(c.minActivePhases())
}

func (c *CurrentController) MaxPower() float64 {
	res := Voltage * c.effectiveMaxCurrent() * float64(c.maxActivePhases())
	if v := c.host.GetVehicle(); v != nil {
		if maxPower, ok := v.OnIdentified().GetMaxPower(); ok {
			return min(maxPower, res)
		}
	}
	return res
}

func (c *CurrentController) EffectiveChargePower() float64 {
	if !c.host.Charging() {
		return 0
	}
	// Zoe hysteresis: vehicle charges below offered current
	if c.chargeCurrents != nil {
		cur := max(c.chargeCurrents[0], c.chargeCurrents[1], c.chargeCurrents[2])
		effectiveCur := min(cur+2.0, c.offeredCurrent)
		return effectiveCur * float64(c.activePhases()) * Voltage
	}
	return c.offeredCurrent * float64(c.activePhases()) * Voltage
}

// --- Read-only accessors for loadpoint ---

func (c *CurrentController) ActivePhases() int    { return c.activePhases() }
func (c *CurrentController) GetPhases() int        { return c.phases }
func (c *CurrentController) GetPhasesConfigured() int { return c.phasesConfigured }
func (c *CurrentController) GetMeasuredPhases() int { return c.measuredPhases }
func (c *CurrentController) GetOfferedCurrent() float64 { return c.offeredCurrent }
func (c *CurrentController) HasPhaseSwitching() bool { return c.hasPhaseSwitching() }
func (c *CurrentController) GetMinCurrent() float64 { return c.minCurrent }
func (c *CurrentController) GetMaxCurrent() float64 { return c.maxCurrent }
func (c *CurrentController) GetEffectiveMinCurrent() float64 { return c.effectiveMinCurrent() }
func (c *CurrentController) GetEffectiveMaxCurrent() float64 { return c.effectiveMaxCurrent() }
func (c *CurrentController) IsChargerUpdateCompleted() bool { return c.chargerUpdateCompleted() }
func (c *CurrentController) IsPhaseSwitchCompleted() bool { return c.phaseSwitchCompleted() }

// --- Setters called by loadpoint ---

func (c *CurrentController) SetPhasesConfigured(phases int) {
	if c.phasesConfigured != phases {
		c.phasesConfigured = phases
		// configured phases are actual phases for non-1p3p charger
		if !c.hasPhaseSwitching() {
			c.setPhases(phases)
		}
	}
}

func (c *CurrentController) SetMinCurrent(current float64) { c.minCurrent = current }
func (c *CurrentController) SetMaxCurrent(current float64) { c.maxCurrent = current }
func (c *CurrentController) SetEnabled(enabled bool)       { c.enabled = enabled }

func (c *CurrentController) SetEnableDelay(d time.Duration)  { c.enableDelay = d }
func (c *CurrentController) SetDisableDelay(d time.Duration) { c.disableDelay = d }

// UpdateChargePower updates the measured charge power from the meter.
func (c *CurrentController) UpdateChargePower(power float64) {
	c.chargePower = power
}

// UpdateChargeCurrents updates the measured per-phase currents from the meter.
func (c *CurrentController) UpdateChargeCurrents(currents []float64) {
	c.chargeCurrents = currents
}

// ResetMeasuredPhases resets measured phases to unknown.
func (c *CurrentController) ResetMeasuredPhases() {
	c.measuredPhases = 0
}

// UpdateMeasuredPhases detects active phases from charge currents.
func (c *CurrentController) UpdateMeasuredPhases() {
	if c.chargeCurrents == nil {
		return
	}
	if c.host.Charging() && c.phaseSwitchCompleted() {
		var phases int
		for _, i := range c.chargeCurrents {
			if i > minActiveCurrent {
				phases++
			}
		}
		if phases >= 1 {
			c.measuredPhases = phases
			c.log.DEBUG.Printf("detected active phases: %dp", phases)
			c.publish("phasesActive", phases)
		}
	}
}

// --- Internal methods ---

func (c *CurrentController) effectiveMinCurrent() float64 {
	var vehicleMin, chargerMin float64
	if v := c.host.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
	}
	if c.currentLimiter != nil {
		if res, _, err := c.currentLimiter.GetMinMaxCurrent(); err == nil {
			chargerMin = res
		}
	}
	switch {
	case max(vehicleMin, chargerMin) == 0:
		return c.minCurrent
	case chargerMin > 0:
		return max(vehicleMin, chargerMin)
	default:
		return max(vehicleMin, c.minCurrent)
	}
}

func (c *CurrentController) effectiveMaxCurrent() float64 {
	maxCurrent := c.maxCurrent
	if v := c.host.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}
	if c.currentLimiter != nil {
		if _, res, err := c.currentLimiter.GetMinMaxCurrent(); err == nil && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}
	return maxCurrent
}

func (c *CurrentController) coarseCurrent() bool {
	if c.chargerEx == nil {
		return true
	}
	if v := c.host.GetVehicle(); v != nil {
		return slices.Contains(v.Features(), api.CoarseCurrent)
	}
	return false
}

func (c *CurrentController) roundedCurrent(current float64) float64 {
	if c.coarseCurrent() {
		current = math.Trunc(current)
	}
	return current
}

func (c *CurrentController) setChargerCurrent(current float64) error {
	var err error
	if c.chargerEx != nil {
		err = c.chargerEx.MaxCurrentMillis(current)
	} else {
		err = c.charger.MaxCurrent(int64(current))
	}

	if err != nil {
		if errors.Is(err, api.ErrAsleep) {
			c.log.DEBUG.Printf("set charge current limit: waking up vehicle")
			if wakeErr := c.host.WakeUpVehicle(); wakeErr != nil {
				return fmt.Errorf("wake-up vehicle: %w", wakeErr)
			}
		}
		return fmt.Errorf("set charge current limit %.3gA: %w", current, err)
	}

	c.log.DEBUG.Printf("set charge current limit: %.3gA", current)
	c.offeredCurrent = current
	return nil
}

// --- Phase management ---

func (c *CurrentController) hasPhaseSwitching() bool {
	return c.phaseSwitcher != nil
}

func (c *CurrentController) getChargerPhysicalPhases() int {
	if c.phaseDescriber != nil {
		return c.phaseDescriber.Phases()
	}
	return 0
}

func (c *CurrentController) getVehiclePhases() int {
	if v := c.host.GetVehicle(); v != nil {
		return v.Phases()
	}
	return 0
}

func expect(phases int) int {
	if phases > 0 {
		return phases
	}
	return unknownPhases
}

func (c *CurrentController) activePhases() int {
	physical := c.phases
	vehicle := c.getVehiclePhases()
	measured := c.measuredPhases
	charger := c.getChargerPhysicalPhases()

	active := min(expect(vehicle), expect(physical), expect(measured), expect(charger))

	// sanity check
	if measured > 0 && active < measured {
		c.log.WARN.Printf("phase mismatch between %dp measured for %dp vehicle and %dp charger", measured, vehicle, physical)
	}

	return active
}

func (c *CurrentController) minActivePhases() int {
	if c.hasPhaseSwitching() || c.phasesConfigured == 1 {
		return 1
	}
	return c.maxActivePhases()
}

func (c *CurrentController) maxActivePhases() int {
	physical := c.phases
	measured := c.measuredPhases
	vehicle := c.getVehiclePhases()
	charger := c.getChargerPhysicalPhases()

	// during 1p or unknown config, 1p measured is not a restriction
	if physical <= 1 || vehicle == 1 || charger == 1 {
		measured = 0
	}

	// if 1p3p supported then assume configured limit or 3p
	if c.hasPhaseSwitching() {
		physical = c.phasesConfigured
	}

	return min(expect(vehicle), expect(physical), expect(measured), expect(charger))
}

func (c *CurrentController) setPhases(phases int) {
	if c.phases != phases {
		c.phases = phases
		c.resetPhaseTimer()
		c.ResetMeasuredPhases()
	}
}

// SetPhases sets the number of enabled phases without modifying the charger.
func (c *CurrentController) SetPhases(phases int) {
	c.setPhases(phases)
}

func (c *CurrentController) scalePhases(phases int) error {
	if c.phaseSwitcher == nil {
		panic("charger does not implement api.PhaseSwitcher")
	}

	if c.phases != phases {
		if err := c.phaseSwitcher.Phases1p3p(phases); err != nil {
			return fmt.Errorf("switch phases: %w", err)
		}

		c.log.DEBUG.Printf("switched phases: %dp", phases)

		// prevent premature measurement of active phases
		c.phasesSwitched = c.clock.Now()

		// update setting and reset timer
		c.setPhases(phases)

		// some vehicles may hang on phase switch
		c.host.StartWakeUpTimer()
	}

	return nil
}

func (c *CurrentController) scalePhasesIfAvailable(phases int) error {
	if c.phasesConfigured != 0 {
		phases = c.phasesConfigured
	}
	if c.hasPhaseSwitching() {
		return c.scalePhases(phases)
	}
	return nil
}

// ScalePhasesRequired returns true if fixed phase configuration differs from current state.
func (c *CurrentController) ScalePhasesRequired() bool {
	return c.hasPhaseSwitching() && c.phasesConfigured != 0 && c.phasesConfigured != c.phases
}

func (c *CurrentController) chargerUpdateCompleted() bool {
	return c.clock.Since(c.chargerSwitched) > chargerSwitchDuration
}

func (c *CurrentController) phaseSwitchCompleted() bool {
	return c.clock.Since(c.phasesSwitched) > phaseSwitchDuration
}

func (c *CurrentController) resetPhaseTimer() {
	if c.phaseTimer.IsZero() {
		return
	}
	c.phaseTimer = time.Time{}
	c.publishTimer(phaseTimer, 0, timerInactive)
}

func (c *CurrentController) publishTimer(name string, delay time.Duration, action string) {
	remaining := max(delay-c.clock.Since(c.phaseTimer), 0)
	c.publish(name+"Action", action)
	c.publish(name+"Remaining", remaining)

	if action == timerInactive {
		c.log.DEBUG.Printf("%s timer %s", name, action)
	} else {
		c.log.DEBUG.Printf("%s %s in %v", name, action, remaining.Round(time.Second))
	}
}

// optimizePhases decides whether to switch phases based on target power.
func (c *CurrentController) optimizePhases(targetPower float64) {
	activePhases := c.activePhases()
	effMinCurrent := c.effectiveMinCurrent()
	effMaxCurrent := c.effectiveMaxCurrent()

	// observed phase state inconsistency
	if measuredPhases := c.measuredPhases; c.phases > 0 && c.phases < measuredPhases {
		if c.chargerUpdateCompleted() && c.phaseSwitchCompleted() {
			c.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", c.phases, measuredPhases)
		}
		c.ResetMeasuredPhases()
	}

	minPowerAtActive := currentToPower(effMinCurrent, activePhases)
	maxPowerAt1p := currentToPower(effMaxCurrent, 1)

	scalable := (targetPower < minPowerAtActive || !c.enabled) &&
		activePhases > 1 && c.phasesConfigured < 3

	// Scale down: target power below minimum at current phase count
	if scalable {
		c.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", targetPower, minPowerAtActive, activePhases)

		if !c.host.Charging() {
			c.phaseTimer = elapsed // scale immediately if not charging
		}
		if c.phaseTimer.IsZero() {
			c.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			c.phaseTimer = c.clock.Now()
		}
		c.publishTimer(phaseTimer, c.disableDelay, phaseScale1p)

		if elapsed := c.clock.Since(c.phaseTimer); elapsed >= c.disableDelay {
			if err := c.scalePhases(1); err != nil {
				c.log.ERROR.Println(err)
			}
			return
		}

		// suppress duplicate log message after timer started
		return
	}

	// Scale up: target power exceeds 1p max AND fits minimum at max phases
	maxPhases := c.maxActivePhases()
	minPowerAtMax := currentToPower(effMinCurrent, maxPhases)

	if activePhases == 1 && targetPower > maxPowerAt1p && targetPower >= minPowerAtMax {
		c.log.DEBUG.Printf("available power %.0fW > %.0fW max 1p threshold", targetPower, maxPowerAt1p)

		if !c.host.Charging() {
			c.phaseTimer = elapsed // scale immediately if not charging
		}
		if c.phaseTimer.IsZero() {
			c.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			c.phaseTimer = c.clock.Now()
		}
		c.publishTimer(phaseTimer, c.enableDelay, phaseScale3p)

		if elapsed := c.clock.Since(c.phaseTimer); elapsed >= c.enableDelay {
			if err := c.scalePhases(maxPhases); err != nil {
				c.log.ERROR.Println(err)
			}
			return
		}

		return
	}

	// No scaling needed — reset timer
	c.resetPhaseTimer()
}

// --- Helpers ---

func powerToCurrent(power float64, phases int) float64 {
	if phases == 0 {
		return 0
	}
	return power / float64(phases) / Voltage
}

func currentToPower(current float64, phases int) float64 {
	return current * float64(phases) * Voltage
}
