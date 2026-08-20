package core

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

var _ site.API = (*Site)(nil)

var (
	ErrBatteryNotConfigured       = errors.New("battery not configured")
	ErrBatteryControlNotAvailable = errors.New("battery control not available")
)

// isConfigurable checks if the meter is configurable
func isConfigurable(ref string) bool {
	dev, _ := config.Meters().ByName(ref)
	_, ok := dev.(config.ConfigurableDevice[api.Meter])
	return ok
}

// filterConfigurable filters configurable meters
func filterConfigurable(ref []string) []string {
	return lo.Filter(ref, func(ref string, _ int) bool {
		return isConfigurable(ref)
	})
}

// Optimize updates the optimizer
func (site *Site) Optimize() {
	go site.optimizerUpdateAsync(0)
}

// GetTitle returns the title
func (site *Site) GetTitle() string {
	site.RLock()
	defer site.RUnlock()
	return site.Title
}

// SetTitle sets the title
func (site *Site) SetTitle(title string) {
	site.Lock()
	defer site.Unlock()

	site.Title = title
	site.publish(keys.SiteTitle, title)
	settings.SetString(keys.Title, title)
}

// GetGridMeterRef returns the GridMeterRef
func (site *Site) GetGridMeterRef() string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.GridMeterRef
}

// SetGridMeterRef sets the GridMeterRef
func (site *Site) SetGridMeterRef(ref string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.GridMeterRef = ref
	settings.SetString(keys.GridMeter, ref)
}

// GetPVMeterRefs returns the PvMeterRef
func (site *Site) GetPVMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.PVMetersRef
}

// SetPVMeterRefs sets the PvMeterRef
func (site *Site) SetPVMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.PVMetersRef = ref
	settings.SetString(keys.PvMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetBatteryMeterRefs returns the BatteryMeterRef
func (site *Site) GetBatteryMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.BatteryMetersRef
}

// SetBatteryMeterRefs sets the BatteryMeterRef
func (site *Site) SetBatteryMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.BatteryMetersRef = ref
	settings.SetString(keys.BatteryMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetAuxMeterRefs returns the AuxMeterRef
func (site *Site) GetAuxMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.AuxMetersRef
}

// SetAuxMeterRefs sets the AuxMeterRef
func (site *Site) SetAuxMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.AuxMetersRef = ref
	settings.SetString(keys.AuxMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetConsumerMeterRefs returns the ConsumerMeterRef
func (site *Site) GetConsumerMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.ConsumerMetersRef
}

// SetConsumerMeterRefs sets the ConsumerMeterRef
func (site *Site) SetConsumerMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.ConsumerMetersRef = ref
	settings.SetString(keys.ConsumerMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetExtMeterRefs returns the ExtMeterRef
func (site *Site) GetExtMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.ExtMetersRef
}

// SetExtMeterRefs sets the ExtMeterRef
func (site *Site) SetExtMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.ExtMetersRef = ref
	settings.SetString(keys.ExtMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetBatterySoc returns the current battery soc
func (site *Site) GetBatterySoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.battery.Soc
}

// GetBatteryMaxDischargePower returns the current battery max discharge power
func (site *Site) GetBatteryMaxDischargePower() *float64 {
	site.RLock()
	defer site.RUnlock()
	if site.batteryMaxDischargePower == nil {
		return nil
	}
	return new(*site.batteryMaxDischargePower)
}

// Loadpoints returns the loadpoints as api interfaces.
// Disabled loadpoints are returned as nil to keep indexes stable.
func (site *Site) Loadpoints() []loadpoint.API {
	return lo.Map(site.loadpoints, func(lp *Loadpoint, _ int) loadpoint.API {
		if lp == nil {
			return nil
		}
		return lp
	})
}

// ActiveLoadpoints yields enabled loadpoints with their stable index
func (site *Site) ActiveLoadpoints() iter.Seq2[int, loadpoint.API] {
	return func(yield func(int, loadpoint.API) bool) {
		for id, lp := range site.loadpoints {
			if lp != nil && !yield(id, lp) {
				return
			}
		}
	}
}

func (site *Site) hasMeters() bool {
	return site.gridMeter != nil || len(site.pvMeters) > 0 || len(site.batteryMeters) > 0 || len(site.auxMeters) > 0 || len(site.extMeters) > 0
}

func (site *Site) IsConfigured() bool {
	return slices.ContainsFunc(site.loadpoints, func(lp *Loadpoint) bool { return lp != nil }) || site.hasMeters()
}

// activeLoadpoints returns the non-disabled loadpoints
func (site *Site) activeLoadpoints() []*Loadpoint {
	return lo.Filter(site.loadpoints, func(lp *Loadpoint, _ int) bool { return lp != nil })
}

// loadpointsAsCircuitDevices returns the loadpoints as circuit devices
func (site *Site) loadpointsAsCircuitDevices() []api.CircuitLoad {
	return lo.Map(site.activeLoadpoints(), func(lp *Loadpoint, _ int) api.CircuitLoad { return lp })
}

// Vehicles returns the site vehicles
func (site *Site) Vehicles() site.Vehicles {
	return &vehicles{log: site.log}
}

// GetCircuit returns the root circuit
func (site *Site) GetCircuit() api.Circuit {
	site.RLock()
	defer site.RUnlock()
	return site.circuit
}

// SetHEMS attaches the configured HEMS to the site and the root circuit
func (site *Site) SetHEMS(hems api.HEMS) {
	site.Lock()
	defer site.Unlock()
	site.hems = hems

	if site.circuit != nil {
		site.circuit.SetHEMS(hems)
	}
}

// GetPrioritySoc returns the PrioritySoc
func (site *Site) GetPrioritySoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.prioritySoc
}

// SetPrioritySoc sets the PrioritySoc
func (site *Site) SetPrioritySoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	if site.bufferSoc != 0 && soc > site.bufferSoc {
		return errors.New("priority soc must be smaller or equal than buffer soc")
	}

	site.log.DEBUG.Println("set priority soc:", soc)

	if site.prioritySoc != soc {
		site.prioritySoc = soc
		settings.SetFloat(keys.PrioritySoc, site.prioritySoc)
		site.publish(keys.PrioritySoc, site.prioritySoc)
	}

	return nil
}

// GetBufferSoc returns the BufferSoc
func (site *Site) GetBufferSoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.bufferSoc
}

// SetBufferSoc sets the BufferSoc
func (site *Site) SetBufferSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	if soc != 0 && soc < site.prioritySoc {
		return errors.New("buffer soc must not be smaller than priority soc")
	}

	if site.bufferStartSoc != 0 && soc > site.bufferStartSoc {
		return errors.New("buffer soc must be smaller or equal than buffer start soc")
	}

	site.log.DEBUG.Println("set buffer soc:", soc)

	if site.bufferSoc != soc {
		site.bufferSoc = soc
		settings.SetFloat(keys.BufferSoc, site.bufferSoc)
		site.publish(keys.BufferSoc, site.bufferSoc)
	}

	return nil
}

// GetBufferStartSoc returns the BufferStartSoc
func (site *Site) GetBufferStartSoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.bufferStartSoc
}

// SetBufferStartSoc sets the BufferStartSoc
func (site *Site) SetBufferStartSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	if soc != 0 && soc < site.bufferSoc {
		return errors.New("buffer start soc must be larger than buffer soc")
	}

	site.log.DEBUG.Println("set buffer start soc:", soc)

	if site.bufferStartSoc != soc {
		site.bufferStartSoc = soc
		settings.SetFloat(keys.BufferStartSoc, site.bufferStartSoc)
		site.publish(keys.BufferStartSoc, site.bufferStartSoc)
	}

	return nil
}

// GetGridPower returns the most recent grid power reading in W (positive = import)
func (site *Site) GetGridPower() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.gridPower
}

// GetResidualPower returns the ResidualPower
func (site *Site) GetResidualPower() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.ResidualPower
}

// SetResidualPower sets the ResidualPower
func (site *Site) SetResidualPower(power float64) error {
	site.log.DEBUG.Println("set residual power:", power)

	site.Lock()
	defer site.Unlock()

	if site.ResidualPower != power {
		site.ResidualPower = power
		settings.SetFloat(keys.ResidualPower, site.ResidualPower)
		site.publish(keys.ResidualPower, site.ResidualPower)
	}

	return nil
}

// GetGridExportLimit returns the static grid export power limit in W (0 = disabled)
func (site *Site) GetGridExportLimit() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.gridExportLimit
}

// SetGridExportLimit sets the static grid export power limit in W (0 = disabled)
func (site *Site) SetGridExportLimit(power float64) error {
	if power < 0 {
		return fmt.Errorf("invalid grid export limit: %g", power)
	}

	site.Lock()
	changed := site.gridExportLimit != power
	if changed {
		site.gridExportLimit = power
	}
	site.Unlock()

	if changed {
		site.log.DEBUG.Println("set grid export limit:", power)
		settings.SetFloat(keys.GridExportLimit, power)
		site.publish(keys.GridExportLimit, power)

		// re-run the optimizer so the new limit takes effect immediately
		go site.optimizerUpdateAsync(0)
	}

	return nil
}

// GetTariff returns the respective tariff if configured or nil
func (site *Site) GetTariff(tariff api.TariffUsage) api.Tariff {
	site.RLock()
	defer site.RUnlock()
	return site.tariffs.Get(tariff)
}

// GetBatteryDischargeControl returns the battery control mode (no discharge only)
func (site *Site) GetBatteryDischargeControl() bool {
	site.RLock()
	defer site.RUnlock()
	return site.batteryDischargeControl
}

// SetBatteryDischargeControl sets the battery control mode (no discharge only)
func (site *Site) SetBatteryDischargeControl(val bool) error {
	site.log.DEBUG.Println("set battery discharge control:", val)

	if !site.hasBatteryControl() {
		return ErrBatteryControlNotAvailable
	}

	site.Lock()
	defer site.Unlock()

	if site.batteryDischargeControl != val {
		site.batteryDischargeControl = val
		settings.SetBool(keys.BatteryDischargeControl, val)
		site.publish(keys.BatteryDischargeControl, val)
	}

	return nil
}

func (site *Site) GetOptimizerManualPA() *float64 {
	site.RLock()
	defer site.RUnlock()
	return site.optimizerManualPA
}

func (site *Site) SetOptimizerManualPA(val *float64) error {
	site.log.DEBUG.Println("set optimizer manual p_a:", printPtr("%.3f", val))

	var changed bool

	site.Lock()
	if !ptrValueEqual(site.optimizerManualPA, val) {
		site.optimizerManualPA = val

		if val == nil {
			settings.SetString(keys.OptimizerManualPA, "")
			site.publish(keys.OptimizerManualPA, nil)
		} else {
			settings.SetFloat(keys.OptimizerManualPA, *val)
			site.publish(keys.OptimizerManualPA, *val)
		}

		changed = true
	}
	site.Unlock()

	if changed {
		site.triggerOptimizer()
	}

	return nil
}

// GetBatteryGridDischarge returns whether the battery may discharge to grid (experimental)
func (site *Site) GetBatteryGridDischarge() bool {
	site.RLock()
	defer site.RUnlock()
	return site.batteryGridDischarge
}

// SetBatteryGridDischarge sets whether the battery may discharge to grid (experimental)
func (site *Site) SetBatteryGridDischarge(val bool) error {
	site.log.DEBUG.Println("set battery grid discharge:", val)

	if !site.hasBatteryControl() {
		return ErrBatteryControlNotAvailable
	}

	site.Lock()
	defer site.Unlock()

	if site.batteryGridDischarge != val {
		site.batteryGridDischarge = val
		settings.SetBool(keys.BatteryGridDischarge, val)
		site.publish(keys.BatteryGridDischarge, val)
	}

	return nil
}

// GetSolarAdjusted returns if the solar forecast is adjusted to real production data
func (site *Site) GetSolarAdjusted() bool {
	site.RLock()
	defer site.RUnlock()
	return site.solarAdjusted
}

// SetSolarAdjusted sets if the solar forecast is adjusted to real production data
func (site *Site) SetSolarAdjusted(val bool) {
	site.log.DEBUG.Println("set solar adjusted:", val)

	site.Lock()
	defer site.Unlock()

	if site.solarAdjusted != val {
		site.solarAdjusted = val
		settings.SetBool(keys.SolarAdjusted, val)
		site.publish(keys.SolarAdjusted, val)
	}
}

func (site *Site) GetBatteryGridChargeLimit() *float64 {
	site.RLock()
	defer site.RUnlock()
	return site.batteryGridChargeLimit
}

func (site *Site) SetBatteryGridChargeLimit(val *float64) error {
	site.log.DEBUG.Println("set grid charge limit:", printPtr("%.1f", val))

	if !site.hasBatteryControl() {
		return ErrBatteryControlNotAvailable
	}

	site.Lock()
	defer site.Unlock()

	if !ptrValueEqual(site.batteryGridChargeLimit, val) {
		site.batteryGridChargeLimit = val

		if val == nil {
			settings.SetString(keys.BatteryGridChargeLimit, "")
			site.publish(keys.BatteryGridChargeLimit, nil)
		} else {
			settings.SetFloat(keys.BatteryGridChargeLimit, *val)
			site.publish(keys.BatteryGridChargeLimit, *val)
		}
	}

	return nil
}

func (site *Site) GetBatteryOptimizerSocGoals() []api.RepeatingPlan {
	site.RLock()
	defer site.RUnlock()
	return site.batteryOptimizerSocGoals
}

func (site *Site) SetBatteryOptimizerSocGoals(goals []api.RepeatingPlan) error {
	site.log.DEBUG.Printf("set battery optimizer soc goals: %+v", goals)

	if !site.hasBatteryControl() {
		return ErrBatteryControlNotAvailable
	}

	for i, g := range goals {
		// inactive or weekday-less goals are ignored by the optimizer, so don't
		// reject them here - matches applyBatterySocGoals and the shared UI, which
		// lets a plan be toggled off or left without weekdays
		if !g.Active || len(g.Weekdays) == 0 {
			continue
		}
		if err := validateBatteryOptimizerSocGoal(g); err != nil {
			return fmt.Errorf("battery optimizer soc goal %d: %w", i+1, err)
		}
	}

	var changed bool

	site.Lock()
	if !slices.EqualFunc(site.batteryOptimizerSocGoals, goals, repeatingPlanEqual) {
		site.batteryOptimizerSocGoals = goals

		if len(goals) == 0 {
			settings.SetString(keys.BatteryOptimizerSocGoals, "")
		} else if err := settings.SetJson(keys.BatteryOptimizerSocGoals, goals); err != nil {
			site.log.ERROR.Printf("battery optimizer soc goals: %v", err)
		}
		site.publish(keys.BatteryOptimizerSocGoals, goals)

		changed = true
	}
	site.Unlock()

	if changed {
		site.triggerOptimizer()
	}

	return nil
}

// validateBatteryOptimizerSocGoal checks a single reserve goal. Time is
// meaningless without its zone, so an explicit valid IANA timezone is required.
func validateBatteryOptimizerSocGoal(g api.RepeatingPlan) error {
	if g.Soc <= 0 || g.Soc > 100 {
		return errors.New("soc must be greater than 0 and at most 100")
	}
	if _, err := time.Parse("15:04", g.Time); err != nil {
		return errors.New("time must use HH:MM format")
	}
	if g.Tz == "" {
		return errors.New("timezone is required")
	}
	if _, err := time.LoadLocation(g.Tz); err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}
	if len(g.Weekdays) == 0 {
		return errors.New("at least one weekday is required")
	}
	for _, d := range g.Weekdays {
		if d < 0 || d > 6 {
			return errors.New("weekdays must be 0..6 (Sunday..Saturday)")
		}
	}
	return nil
}

// repeatingPlanEqual compares two repeating plans by value (Weekdays is a slice).
func repeatingPlanEqual(a, b api.RepeatingPlan) bool {
	return a.Time == b.Time && a.Tz == b.Tz && a.Soc == b.Soc &&
		a.Active == b.Active && slices.Equal(a.Weekdays, b.Weekdays)
}

// loadBatteryOptimizerSocGoals reads the persisted goals; returns (nil, err)
// when absent or malformed so callers can simply skip it.
func loadBatteryOptimizerSocGoals() ([]api.RepeatingPlan, error) {
	var goals []api.RepeatingPlan
	if err := settings.Json(keys.BatteryOptimizerSocGoals, &goals); err != nil {
		return nil, err
	}
	return goals, nil
}

// GetOptimizerChargingStrategy returns the optimizer grid charging strategy,
// falling back to the default when unset.
func (site *Site) GetOptimizerChargingStrategy() string {
	site.RLock()
	defer site.RUnlock()
	if site.optimizerChargingStrategy == "" {
		return defaultOptimizerChargingStrategy
	}
	return site.optimizerChargingStrategy
}

// SetOptimizerChargingStrategy validates and persists the optimizer grid
// charging strategy and re-runs the optimizer when it changes.
func (site *Site) SetOptimizerChargingStrategy(strategy string) error {
	if !slices.Contains(optimizerChargingStrategies, strategy) {
		return fmt.Errorf("invalid optimizer charging strategy: %s", strategy)
	}

	site.Lock()
	changed := site.optimizerChargingStrategy != strategy
	if changed {
		site.optimizerChargingStrategy = strategy
	}
	site.Unlock()

	if changed {
		site.log.DEBUG.Println("set optimizer charging strategy:", strategy)
		settings.SetString(keys.OptimizerChargingStrategy, strategy)
		site.publish(keys.OptimizerChargingStrategy, strategy)

		// re-run the optimizer so the new strategy takes effect immediately
		go site.optimizerUpdateAsync(0)
	}

	return nil
}

// GetBatteryMode returns the battery mode
func (site *Site) GetBatteryMode() api.BatteryMode {
	site.RLock()
	defer site.RUnlock()
	return site.batteryMode
}

// GetBatteryModeExternal returns the external battery mode
func (site *Site) GetBatteryModeExternal() api.BatteryMode {
	site.RLock()
	defer site.RUnlock()
	return site.batteryModeExternal
}

// SetBatteryModeExternal sets the external battery mode
func (site *Site) SetBatteryModeExternal(mode api.BatteryMode) error {
	site.log.DEBUG.Printf("set external battery mode: %s", mode.String())

	if !site.hasBatteryControl() {
		return ErrBatteryControlNotAvailable
	}

	site.Lock()
	defer site.Unlock()

	disable := mode == api.BatteryUnknown

	if mode != site.batteryModeExternal {
		site.batteryModeExternal = mode
		site.publish(keys.BatteryModeExternal, mode)

		// start watchdog if not running
		if !disable && site.batteryModeExternalTimer.IsZero() {
			go func() {
				for range time.Tick(time.Second) {
					if site.batteryModeWatchdogExpired() {
						return
					}
				}
			}()
		}
	}

	// reset timer
	if !disable {
		site.batteryModeExternalTimer = time.Now()
	}

	return nil
}

func (site *Site) batteryModeWatchdogExpired() bool {
	site.RLock()
	elapsed := time.Since(site.batteryModeExternalTimer)
	site.RUnlock()

	if elapsed > time.Minute && !site.batteryModeExternalTimer.IsZero() {
		site.SetBatteryModeExternal(api.BatteryUnknown)
		return true
	}

	return false
}
