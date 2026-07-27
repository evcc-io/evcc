package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/prioritizer"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/telemetry"
	"github.com/samber/lo"
	"github.com/smallnest/chanx"
	"golang.org/x/sync/errgroup"
)

const standbyPower = 10 // consider less than 10W as charger in standby

// updater abstracts the Loadpoint implementation for testing
type updater interface {
	loadpoint.API
	Update(sitePower, batteryBoostPower float64, consumption, feedin api.Rates, batteryBuffered, batteryStart bool, greenShare float64, effectivePrice, effectiveCo2 *float64, dim *bool)
}

var _ site.API = (*Site)(nil)

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	valueChan    chan<- util.Param      // client push messages
	pushChan     chan<- messenger.Event // notification events
	lpUpdateChan chan *Loadpoint

	sync.RWMutex
	log *util.Logger

	// configuration
	Title         string       `mapstructure:"title"`         // UI title
	Voltage       float64      `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower float64      `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Meters        MetersConfig `mapstructure:"meters"`        // Meter references

	// meters
	circuit        api.Circuit                // Circuit
	hems           api.HEMS                   // HEMS (set by configureHEMS at boot)
	gridMeter      api.Meter                  // Grid usage meter
	pvMeters       []config.Device[api.Meter] // PV generation meters
	batteryMeters  []config.Device[api.Meter] // Battery charging meters
	extMeters      []config.Device[api.Meter] // External meters - for monitoring only
	auxMeters      []config.Device[api.Meter] // Auxiliary meters
	consumerMeters []config.Device[api.Meter] // Consumer meters

	// last applied HEMS state, nil until applied or after a failed attempt
	dimmed         *bool
	curtailPercent *int

	// battery settings
	prioritySoc             float64  // prefer battery up to this Soc
	bufferSoc               float64  // continue charging on battery above this Soc
	bufferStartSoc          float64  // start charging on battery above this Soc
	batteryDischargeControl bool     // prevent battery discharge for fast and planned charging
	batteryGridChargeLimit  *float64 // grid charging limit
	batteryGridDischarge    bool     // allow battery discharge to grid (experimental)

	// forecast settings
	solarAdjusted bool // adjust solar forecast to real production data

	// optimizer settings
	optimizerChargingStrategy string // optimizer grid charging strategy

	loadpoints  []*Loadpoint             // Loadpoints
	tariffs     *tariff.Tariffs          // Tariffs
	coordinator *coordinator.Coordinator // Vehicles
	prioritizer *prioritizer.Prioritizer // Power budgets
	stats       *Stats                   // Stats

	collectors map[string]*metrics.Collector // keyed by meter ref
	tariffSlot time.Time                     // last persisted tariff slot

	// cached state
	gridPower                float64                     // Grid power
	pvPower                  float64                     // PV power
	excessDCPower            float64                     // PV excess DC charge power (hybrid only)
	auxPower                 float64                     // Aux power
	battery                  types.BatteryState          // Battery cached and published state
	batteryMode              api.BatteryMode             // Battery mode (runtime only, not persisted)
	batteryModeExternal      api.BatteryMode             // Battery mode (external, runtime only, not persisted)
	batteryModeExternalTimer time.Time                   // Battery mode timer for external control
	batterySuggestions       map[string]types.Suggestion // Optimizer suggestions by battery meter name
	loadpointSuggestions     map[int]types.Suggestion    // Optimizer suggestions by loadpoint id
	suggestionActions        map[string]string           // last notified actionable optimizer action by device key
}

// MetersConfig contains the site's meter configuration
type MetersConfig struct {
	GridMeterRef      string   `mapstructure:"grid"`     // Grid usage meter
	PVMetersRef       []string `mapstructure:"pv"`       // PV meter
	BatteryMetersRef  []string `mapstructure:"battery"`  // Battery charging meter
	ExtMetersRef      []string `mapstructure:"ext"`      // Meters used only for monitoring
	AuxMetersRef      []string `mapstructure:"aux"`      // Auxiliary meters
	ConsumerMetersRef []string `mapstructure:"consumer"` // Consumer meters
}

// NewSiteFromConfig creates a new site
func NewSiteFromConfig(other map[string]any) (*Site, error) {
	site := NewSite()

	// TODO remove
	if err := util.DecodeOther(other, site); err != nil {
		return nil, err
	}

	// add meters from config
	site.restoreMetersAndTitle()

	// TODO title
	Voltage = site.Voltage

	return site, nil
}

func (site *Site) Boot(log *util.Logger, loadpoints []*Loadpoint, tariffs *tariff.Tariffs) error {
	site.loadpoints = loadpoints
	site.tariffs = tariffs

	handler := config.Vehicles()
	site.coordinator = coordinator.New(log, config.Instances(handler.Devices()))
	handler.Subscribe(site.updateVehicles)

	site.prioritizer = prioritizer.New(log)
	site.stats = NewStats()

	me, err := metrics.NewCollector(metrics.Home, metrics.Home, metrics.Home)
	if err != nil {
		return err
	}
	site.collectors[metrics.Home] = me

	// reload history in the UI on each persisted 15min slot instead of polling
	metrics.OnPersist = func(slot time.Time) { site.publish(keys.HistoryUpdated, slot) }

	// upload telemetry on shutdown
	if telemetry.Enabled() {
		shutdown.Register(func() {
			telemetry.Persist(log)
		})
	}

	tariff := site.GetTariff(api.TariffUsagePlanner)

	// give loadpoints access to vehicles and database
	for _, lp := range loadpoints {
		lp.coordinator = coordinator.NewAdapter(lp, site.coordinator)
		lp.planner = planner.New(lp.log, tariff)

		if db.Instance != nil {
			var err error
			if lp.db, err = session.NewStore(lp.GetTitle(), db.Instance); err != nil {
				return err
			}
			// Fix any dangling history
			if err := lp.db.ClosePendingSessionsInHistory(lp.chargeMeterTotal()); err != nil {
				return err
			}

			// NOTE: this requires stopSession to respect async access
			shutdown.Register(lp.stopSession)
		}
	}

	// circuit
	if c := circuit.Root(); c != nil {
		site.circuit = c
	}

	// grid meter
	if site.Meters.GridMeterRef != "" {
		dev, err := config.Meters().ByName(site.Meters.GridMeterRef)
		if err != nil {
			return err
		}

		site.gridMeter = dev.Instance()
		if site.gridMeter == nil {
			return errors.New("missing grid meter instance")
		}

		me, err := metrics.NewCollector(metrics.Grid, site.Meters.GridMeterRef, metrics.Grid)
		if err != nil {
			return err
		}
		site.collectors[site.Meters.GridMeterRef] = me
	}

	// multiple pv
	for _, ref := range site.Meters.PVMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return err
		}
		site.pvMeters = append(site.pvMeters, dev)

		// energy collector (for history persistence and forecast scaling)
		me, err := metrics.NewCollector(metrics.PV, ref, deviceTitleOrName(dev))
		if err != nil {
			return err
		}
		site.collectors[ref] = me
	}

	// solar forecast collector (mirrors PV history shape, used for scale lookup)
	fc, err := metrics.NewCollector(metrics.Forecast, metrics.Forecast, metrics.Forecast)
	if err != nil {
		return err
	}
	site.collectors[metrics.Forecast] = fc

	// temperature forecast collector (populated when TariffUsageTemperature is configured)
	tc, err := metrics.NewCollector(metrics.Temperature, metrics.Temperature, metrics.Temperature)
	if err != nil {
		return err
	}
	site.collectors[metrics.Temperature] = tc

	// multiple batteries
	for _, ref := range site.Meters.BatteryMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return err
		}
		site.batteryMeters = append(site.batteryMeters, dev)

		me, err := metrics.NewCollector(metrics.Battery, ref, deviceTitleOrName(dev))
		if err != nil {
			return err
		}
		site.collectors[ref] = me
	}

	// additional meters used only for monitoring
	for _, ref := range site.Meters.ExtMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return err
		}
		site.extMeters = append(site.extMeters, dev)

		me, err := metrics.NewCollector(metrics.Meter, ref, deviceTitleOrName(dev))
		if err != nil {
			return err
		}
		site.collectors[ref] = me
	}

	// auxiliary meters (consumers)
	for _, ref := range site.Meters.AuxMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return err
		}
		site.auxMeters = append(site.auxMeters, dev)

		me, err := metrics.NewCollector(metrics.Consumer, ref, deviceTitleOrName(dev))
		if err != nil {
			return err
		}
		site.collectors[ref] = me
	}

	// consumer meters
	for _, ref := range site.Meters.ConsumerMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return err
		}
		site.consumerMeters = append(site.consumerMeters, dev)

		me, err := metrics.NewCollector(metrics.Consumer, ref, deviceTitleOrName(dev))
		if err != nil {
			return err
		}
		site.collectors[ref] = me
	}

	// revert battery mode on shutdown
	shutdown.Register(func() {
		if mode := site.GetBatteryMode(); batteryModeModified(mode) {
			if err := site.applyBatteryMode(api.BatteryNormal); err != nil {
				site.log.ERROR.Println("battery mode:", err)
			}
		}
	})

	return nil
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	site := &Site{
		log:        util.NewLogger("site"),
		Voltage:    230, // V
		collectors: make(map[string]*metrics.Collector),
	}

	return site
}

// restoreMetersAndTitle restores site meter configuration
func (site *Site) restoreMetersAndTitle() {
	if testing.Testing() {
		return
	}
	if v, err := settings.String(keys.Title); err == nil {
		site.Title = v
	}
	if v, err := settings.String(keys.GridMeter); err == nil && v != "" {
		site.Meters.GridMeterRef = v
	}
	if v, err := settings.String(keys.PvMeters); err == nil && v != "" {
		site.Meters.PVMetersRef = append(site.Meters.PVMetersRef, filterConfigurable(strings.Split(v, ","))...)
	}
	if v, err := settings.String(keys.BatteryMeters); err == nil && v != "" {
		site.Meters.BatteryMetersRef = append(site.Meters.BatteryMetersRef, filterConfigurable(strings.Split(v, ","))...)
	}
	if v, err := settings.String(keys.ExtMeters); err == nil && v != "" {
		site.Meters.ExtMetersRef = append(site.Meters.ExtMetersRef, filterConfigurable(strings.Split(v, ","))...)
	}
	if v, err := settings.String(keys.AuxMeters); err == nil && v != "" {
		site.Meters.AuxMetersRef = append(site.Meters.AuxMetersRef, filterConfigurable(strings.Split(v, ","))...)
	}
	if v, err := settings.String(keys.ConsumerMeters); err == nil && v != "" {
		site.Meters.ConsumerMetersRef = append(site.Meters.ConsumerMetersRef, filterConfigurable(strings.Split(v, ","))...)
	}
}

// restoreSettings restores site settings
func (site *Site) restoreSettings() error {
	if testing.Testing() {
		return nil
	}
	if v, err := settings.Float(keys.BufferSoc); err == nil {
		if err := site.SetBufferSoc(v); err != nil && !errors.Is(err, ErrBatteryNotConfigured) {
			return err
		}
	}
	if v, err := settings.Float(keys.BufferStartSoc); err == nil {
		if err := site.SetBufferStartSoc(v); err != nil && !errors.Is(err, ErrBatteryNotConfigured) {
			return err
		}
	}
	if v, err := settings.Float(keys.PrioritySoc); err == nil {
		if err := site.SetPrioritySoc(v); err != nil && !errors.Is(err, ErrBatteryNotConfigured) {
			return err
		}
	}
	if v, err := settings.Bool(keys.BatteryDischargeControl); err == nil {
		if err := site.SetBatteryDischargeControl(v); err != nil && !errors.Is(err, ErrBatteryControlNotAvailable) {
			return err
		}
	}
	if v, err := settings.Bool(keys.BatteryGridDischarge); err == nil {
		if err := site.SetBatteryGridDischarge(v); err != nil && !errors.Is(err, ErrBatteryControlNotAvailable) {
			return err
		}
	}
	if v, err := settings.Float(keys.ResidualPower); err == nil {
		if err := site.SetResidualPower(v); err != nil {
			return err
		}
	}
	if v, err := settings.Float(keys.BatteryGridChargeLimit); err == nil {
		if err := site.SetBatteryGridChargeLimit(&v); err != nil && !errors.Is(err, ErrBatteryControlNotAvailable) {
			return err
		}
	}
	if v, err := settings.Bool(keys.SolarAdjusted); err == nil {
		site.SetSolarAdjusted(v)
	}
	if v, err := settings.String(keys.OptimizerChargingStrategy); err == nil && v != "" {
		if err := site.SetOptimizerChargingStrategy(v); err != nil {
			site.log.WARN.Printf("optimizer charging strategy: %v", err)
		}
	}
	site.publish(keys.OptimizerChargingStrategy, site.GetOptimizerChargingStrategy())
	site.publish(keys.OptimizerChargingStrategies, optimizerChargingStrategies)

	// drop legacy accumulator-based forecast settings (now stored via metrics collector)
	settings.Delete("solarAccForecast")
	settings.Delete("solarAccYield")
	settings.Delete("solarAccDay")

	return nil
}

func meterCapabilities(name string, meter any) string {
	power := api.HasCap[api.Meter](meter)

	if !power {
		panic("not a meter: " + name)
	}

	energy := api.HasCap[api.MeterEnergy](meter)
	currents := api.HasCap[api.PhaseCurrents](meter)

	name += ":"
	return fmt.Sprintf("    %-10s power %s energy %s currents %s",
		name,
		presence[power],
		presence[energy],
		presence[currents],
	)
}

// DumpConfig site configuration
func (site *Site) DumpConfig() {
	// verify vehicle detection
	if vehicles := site.Vehicles().Instances(); len(vehicles) > 1 {
		for _, v := range vehicles {
			if !api.HasCap[api.ChargeState](v) && len(v.Identifiers()) == 0 {
				site.log.INFO.Printf("vehicle '%s' does not support automatic detection", v.GetTitle())
			}
		}
	}

	site.log.INFO.Println("site config:")
	site.log.INFO.Printf("  meters:      grid %s pv %s battery %s",
		presence[site.gridMeter != nil],
		presence[len(site.pvMeters) > 0],
		presence[len(site.batteryMeters) > 0],
	)

	if site.gridMeter != nil {
		site.log.INFO.Println(meterCapabilities("grid", site.gridMeter))
	}

	if len(site.pvMeters) > 0 {
		for i, pv := range site.pvMeters {
			site.log.INFO.Println(meterCapabilities(fmt.Sprintf("pv %d", i+1), pv.Instance()))
		}
	}

	if len(site.batteryMeters) > 0 {
		for i, dev := range site.batteryMeters {
			battery := dev.Instance()
			isBattery := api.HasCap[api.Battery](battery)
			hasCapacity := api.HasCap[api.BatteryCapacity](battery)

			site.log.INFO.Println(
				meterCapabilities(fmt.Sprintf("battery %d", i+1), battery),
				fmt.Sprintf("soc %s capacity %s", presence[isBattery], presence[hasCapacity]),
			)
		}
	}

	if vehicles := site.Vehicles().Instances(); len(vehicles) > 0 {
		site.log.INFO.Println("  vehicles:")

		for i, v := range vehicles {
			_, rng := api.Cap[api.VehicleRange](v)
			_, finish := api.Cap[api.VehicleFinishTimer](v)
			_, status := api.Cap[api.ChargeState](v)
			_, climate := api.Cap[api.VehicleClimater](v)
			_, wakeup := api.Cap[api.Resurrector](v)
			site.log.INFO.Printf("    vehicle %d: range %s finish %s status %s climate %s wakeup %s",
				i+1, presence[rng], presence[finish], presence[status], presence[climate], presence[wakeup],
			)
		}
	}

	site.log.INFO.Println("  tariffs:")
	trf := func(u api.TariffUsage) string {
		if t := site.GetTariff(u); t != nil {
			return t.Type().String()
		}
		return presence[false]
	}
	site.log.INFO.Printf("    grid:      %s", trf(api.TariffUsageGrid))
	site.log.INFO.Printf("    feed-in:   %s", trf(api.TariffUsageFeedIn))
	site.log.INFO.Printf("    co2:       %s", presence[site.GetTariff(api.TariffUsageCo2) != nil])
	site.log.INFO.Printf("    solar:     %s", presence[site.GetTariff(api.TariffUsageSolar) != nil])

	for i, lp := range site.loadpoints {
		lp.log.INFO.Printf("loadpoint %d:", i+1)
		lp.log.INFO.Printf("  mode:        %s", lp.GetMode())

		_, power := api.Cap[api.Meter](lp.charger)
		_, energy := api.Cap[api.MeterEnergy](lp.charger)
		_, currents := api.Cap[api.PhaseCurrents](lp.charger)
		_, phases := api.Cap[api.PhaseSwitcher](lp.charger)
		_, wakeup := api.Cap[api.Resurrector](lp.charger)

		lp.log.INFO.Printf("  charger:     power %s energy %s currents %s phases %s wakeup %s",
			presence[power],
			presence[energy],
			presence[currents],
			presence[phases],
			presence[wakeup],
		)

		lp.log.INFO.Printf("  meters:      charge %s", presence[lp.HasChargeMeter()])

		if lp.HasChargeMeter() {
			lp.log.INFO.Println(meterCapabilities("charge", lp.chargeMeter))
		}
	}
}

// publish sends values to UI and databases
func (site *Site) publish(key string, val any) {
	// test helper
	if site.valueChan == nil {
		return
	}

	site.valueChan <- util.Param{Key: key, Val: val}
}

// publish sends values to UI and databases
func (site *Site) Publish(key string, val any) {
	site.publish(key, val)
}

// publishLoadpoint sends a value into the given loadpoint's state
func (site *Site) publishLoadpoint(id int, key string, val any) {
	// test helper
	if site.valueChan == nil {
		return
	}

	site.valueChan <- util.Param{Loadpoint: &id, Key: key, Val: val}
}

// clearPlanLocks clears locked plan goals for all loadpoints
func (site *Site) clearPlanLocks() {
	for _, lp := range site.Loadpoints() {
		lp.ClearPlanLock()
	}
}

func (site *Site) collectMeters(key string, meters []config.Device[api.Meter]) []types.Measurement {
	mm := make([]types.Measurement, len(meters))

	fun := func(i int, dev config.Device[api.Meter]) {
		meter := dev.Instance()

		props := deviceProperties(dev)
		mm[i] = types.Measurement{
			Name:  dev.Config().Name,
			Title: props.Title,
			Icon:  props.Icon,
		}

		// power
		var b bytes.Buffer
		power, err := backoff.RetryWithData(meter.CurrentPower, modbus.Backoff())
		if err == nil {
			mm[i].Power = power
			site.log.DEBUG.Printf("%s %d power: %.0fW", key, i+1, power)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			if b.Len() > 0 {
				site.log.ERROR.Println("\n" + b.String())
			}
			site.log.ERROR.Printf("%s %d power: %v", key, i+1, err)
		}

		// energy (production); ignore spurious zero readings (NaN-derived or nightly reset, #30950)
		if m, ok := api.Cap[api.MeterEnergy](meter); ok {
			if f, err := nonZeroEnergy(m.TotalEnergy()); err == nil {
				mm[i].Energy = new(f)
			} else if !errors.Is(err, api.ErrNotAvailable) {
				site.log.ERROR.Printf("%s %d energy: %v", key, i+1, err)
			}
		}

		// return energy (export); ignore spurious zero readings as above
		if m, ok := api.Cap[api.MeterReturnEnergy](meter); ok {
			if f, err := nonZeroEnergy(m.ReturnEnergy()); err == nil {
				mm[i].ReturnEnergy = new(f)
			} else if !errors.Is(err, api.ErrNotAvailable) {
				site.log.ERROR.Printf("%s %d return energy: %v", key, i+1, err)
			}
		}
	}

	var wg sync.WaitGroup

	for i, meter := range meters {
		wg.Go(func() {
			fun(i, meter)
		})
	}
	wg.Wait()

	return mm
}

// updatePvMeters updates pv meters. All measurements are optional.
func (site *Site) updatePvMeters() {
	if len(site.pvMeters) == 0 {
		return
	}

	mm := site.collectMeters("pv", site.pvMeters)

	for i, dev := range site.pvMeters {
		meter := dev.Instance()

		power := mm[i].Power
		if power < -500 {
			site.log.WARN.Printf("pv %d power: %.0fW is negative - check configuration if sign is correct", i+1, power)
		}

		if m, ok := api.Cap[api.MaxACPowerGetter](meter); ok {
			if dc := power - m.MaxACPower(); dc > 0 && power > 0 {
				mm[i].ExcessDCPower = dc
				site.log.DEBUG.Printf("pv %d excess DC: %.0fW", i+1, dc)
			}
		}
	}

	site.pvPower = lo.SumBy(mm, func(m types.Measurement) float64 {
		return max(0, m.Power)
	})
	site.excessDCPower = lo.SumBy(mm, func(m types.Measurement) float64 {
		return math.Abs(m.ExcessDCPower)
	})
	totalEnergy := lo.SumBy(mm, func(m types.Measurement) float64 {
		if m.Energy == nil {
			return 0
		}
		return *m.Energy
	})

	if len(site.pvMeters) > 1 {
		var excessStr string
		if site.excessDCPower > 0 {
			excessStr = fmt.Sprintf(" (includes %.0fW excess DC)", site.excessDCPower)
		}

		site.log.DEBUG.Printf("pv power: %.0fW"+excessStr, site.pvPower)
	}

	site.publish(keys.PvPower, site.pvPower)
	site.publish(keys.PvEnergy, totalEnergy)
	site.publish(keys.Pv, mm)

	// persist per-meter PV energy slots (used for history and forecast scaling)
	for i, dev := range site.pvMeters {
		c := site.collectors[dev.Config().Name]
		if err := c.AddEnergy(mm[i].Energy, mm[i].ReturnEnergy, mm[i].Power); err != nil {
			site.log.ERROR.Printf("persist pv %d energy: %v", i+1, err)
		}
	}
}

// updateBatteryMeters updates battery meters
func (site *Site) updateBatteryMeters() {
	if len(site.batteryMeters) == 0 {
		return
	}

	mm := site.collectMeters("battery", site.batteryMeters)

	for i, dev := range site.batteryMeters {
		meter := dev.Instance()

		// battery soc and capacity
		if m, ok := api.Cap[api.Battery](meter); ok {
			batSoc, err := soc.Guard(m.Soc())
			if err == nil {
				mm[i].Soc = new(batSoc)

				if bc, ok := api.Cap[api.BatteryCapacity](meter); ok {
					mm[i].Capacity = new(bc.Capacity())
				}

				site.log.DEBUG.Printf("battery %d soc: %.0f%%", i+1, batSoc)
			} else {
				site.log.ERROR.Printf("battery %d soc: %v", i+1, err)
			}
		}

		_, controllable := api.Cap[api.BatteryController](meter)
		mm[i].Controllable = new(controllable)
	}

	// retain the last known soc when every battery read failed this cycle, so a
	// transient meter error does not report the pack as empty (0%)
	if lo.EveryBy(mm, func(m types.Measurement) bool { return m.Soc == nil }) {
		site.log.WARN.Printf("battery soc: read failed, keeping last %.0f%%", site.battery.Soc)
	} else {
		var batterySocAcc float64
		var totalCapacity float64

		if lo.SomeBy(mm, func(m types.Measurement) bool { return m.Capacity == nil || *m.Capacity <= 0 }) {
			// any capacity is missing
			batterySocAcc = sumOfSocs(mm)
			totalCapacity = float64(len(site.batteryMeters))
		} else {
			// all capacities available - weigh soc by capacity
			batterySocAcc = weightedSumOfSocs(mm)
			totalCapacity = lo.SumBy(mm, func(m types.Measurement) float64 { return *m.Capacity })
		}

		site.battery.Soc = math.Min(100, batterySocAcc/totalCapacity)
		site.battery.Capacity = totalCapacity
	}

	site.battery.Power = lo.SumBy(mm, func(m types.Measurement) float64 {
		return m.Power
	})
	site.battery.Energy = lo.SumBy(mm, func(m types.Measurement) float64 {
		if m.Energy == nil {
			return 0
		}
		return *m.Energy
	})

	if len(site.batteryMeters) > 1 {
		site.log.DEBUG.Printf("battery power: %.0fW", site.battery.Power)
		site.log.DEBUG.Printf("battery soc: %.0f%%", math.Round(site.battery.Soc))
	}

	site.battery.Devices = mm

	// accumulate per-battery energy (charging = import, discharging = export — from battery POV toward grid root)
	for i, dev := range site.batteryMeters {
		ref := dev.Config().Name
		c, ok := site.collectors[ref]
		if !ok {
			continue
		}
		if err := c.AddEnergy(mm[i].ReturnEnergy, mm[i].Energy, -mm[i].Power); err != nil {
			site.log.ERROR.Printf("persist battery %d energy: %v", i+1, err)
		}
		if mm[i].Soc != nil {
			c.SetSocTemp(*mm[i].Soc, false)
		}
	}

	site.publishBattery()
}

// publishBattery applies the optimizer suggestions and publishes the battery state
func (site *Site) publishBattery() {
	for i, d := range site.battery.Devices {
		site.battery.Devices[i].Suggestion = site.batterySuggestion(d.Name)
	}

	site.publish(keys.Battery, site.battery)
}

func sumOfSocs(mm []types.Measurement) float64 {
	return lo.SumBy(mm, func(m types.Measurement) float64 {
		if m.Soc == nil {
			return 0
		}
		return *m.Soc
	})
}

func weightedSumOfSocs(mm []types.Measurement) float64 {
	return lo.SumBy(mm, func(m types.Measurement) float64 {
		if m.Soc == nil {
			return 0
		}
		// weigh soc by capacity
		return *m.Soc * *m.Capacity
	})
}

// addMeterEnergy persists per-meter energy (positive power = import).
func (site *Site) addMeterEnergy(meters []config.Device[api.Meter], mm []types.Measurement) {
	for i, dev := range meters {
		ref := dev.Config().Name
		c, ok := site.collectors[ref]
		if !ok {
			continue
		}
		if err := c.AddEnergy(mm[i].Energy, mm[i].ReturnEnergy, mm[i].Power); err != nil {
			site.log.ERROR.Printf("persist meter %s energy: %v", ref, err)
		}
	}
}

// updateAuxMeters updates aux meters
func (site *Site) updateAuxMeters() {
	if len(site.auxMeters) == 0 {
		return
	}

	mm := site.collectMeters("aux", site.auxMeters)
	site.auxPower = lo.SumBy(mm, func(m types.Measurement) float64 {
		return m.Power
	})

	if len(site.auxMeters) > 1 {
		site.log.DEBUG.Printf("aux power: %.0fW", site.auxPower)
	}

	site.addMeterEnergy(site.auxMeters, mm)

	site.publish(keys.AuxPower, site.auxPower)
	site.publish(keys.Aux, mm)
}

// updateConsumerMeters updates consumer meters
func (site *Site) updateConsumerMeters() {
	if len(site.consumerMeters) == 0 {
		return
	}

	mm := site.collectMeters("consumer", site.consumerMeters)

	site.addMeterEnergy(site.consumerMeters, mm)

	site.publish(keys.Consumers, mm)
}

// updateExtMeters updates ext meters
func (site *Site) updateExtMeters() {
	if len(site.extMeters) == 0 {
		return
	}

	mm := site.collectMeters("ext", site.extMeters)

	site.addMeterEnergy(site.extMeters, mm)

	site.publish(keys.Ext, mm)
}

// updateGridMeter updates grid meter
func (site *Site) updateGridMeter() error {
	if site.gridMeter == nil {
		return nil
	}

	mm := types.Measurement{Name: site.Meters.GridMeterRef}

	if res, err := backoff.RetryWithData(site.gridMeter.CurrentPower, modbus.Backoff()); err == nil {
		mm.Power = res
		site.gridPower = res
		site.log.DEBUG.Printf("grid power: %.0fW", res)
	} else if !errors.Is(err, api.ErrNotAvailable) {
		return fmt.Errorf("grid power: %v", err)
	}

	// grid phase currents (signed)
	if phaseMeter, ok := api.Cap[api.PhaseCurrents](site.gridMeter); ok {
		// grid phase powers
		var p1, p2, p3 float64
		if phaseMeter, ok := api.Cap[api.PhasePowers](site.gridMeter); ok {
			var err error // phases needed for signed currents
			if p1, p2, p3, err = phaseMeter.Powers(); err == nil {
				mm.Powers = []float64{p1, p2, p3}
				site.log.DEBUG.Printf("grid powers: %.0fW", mm.Powers)
			} else if !errors.Is(err, api.ErrNotAvailable) {
				site.log.ERROR.Printf("grid powers: %v", err)
			}
		}

		if i1, i2, i3, err := phaseMeter.Currents(); err == nil {
			mm.Currents = []float64{util.SignFromPower(i1, p1), util.SignFromPower(i2, p2), util.SignFromPower(i3, p3)}
			site.log.DEBUG.Printf("grid currents: %.3gA", mm.Currents)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			site.log.ERROR.Printf("grid currents: %v", err)
		}
	}

	// grid energy (import); nil when the device has no MeterEnergy capability or the read fails
	// ignore spurious zero readings (NaN-derived or nightly reset, #30950)
	if energyMeter, ok := api.Cap[api.MeterEnergy](site.gridMeter); ok {
		if f, err := nonZeroEnergy(energyMeter.TotalEnergy()); err == nil {
			mm.Energy = &f
		} else if !errors.Is(err, api.ErrNotAvailable) {
			site.log.ERROR.Printf("grid energy: %v", err)
		}
	}

	// grid return energy (export); nil when the device has no MeterReturnEnergy capability or the read fails
	// ignore spurious zero readings as above
	if returnEnergyMeter, ok := api.Cap[api.MeterReturnEnergy](site.gridMeter); ok {
		if f, err := nonZeroEnergy(returnEnergyMeter.ReturnEnergy()); err == nil {
			mm.ReturnEnergy = &f
		} else if !errors.Is(err, api.ErrNotAvailable) {
			site.log.ERROR.Printf("grid return energy: %v", err)
		}
	}

	site.collectors[site.Meters.GridMeterRef].AddEnergy(mm.Energy, mm.ReturnEnergy, mm.Power)

	site.publish(keys.Grid, mm)

	return nil
}

func (site *Site) updateMeters() error {
	var eg errgroup.Group

	eg.Go(func() error { site.updatePvMeters(); return nil })
	eg.Go(func() error { site.updateBatteryMeters(); return nil })
	eg.Go(func() error { site.updateAuxMeters(); return nil })
	eg.Go(func() error { site.updateConsumerMeters(); return nil })
	eg.Go(func() error { site.updateExtMeters(); return nil })

	eg.Go(site.updateGridMeter)

	if err := eg.Wait(); err != nil {
		return err
	}

	if sponsor.IsAuthorized() && optimizerEnabled() && time.Since(optimizerUpdated) >= tariff.SlotDuration {
		go site.optimizerUpdateAsync()
	}

	return nil
}

func optimizerEnabled() bool {
	exp, _ := settings.Bool(keys.Experimental)
	opt, _ := settings.Bool(keys.Optimizer)
	return exp && opt
}

// sitePower returns
//   - the net power exported by the site minus a residual margin
//     (negative values mean grid: export, battery: charging
//   - if battery buffer can be used for charging
//   - the adjustment applied to sitePower for battery priority below prioritySoc;
//     adding it back restores the unadjusted site power for a loadpoint that
//     takes priority over the battery (battery boost)
func (site *Site) sitePower(totalChargePower, flexiblePower float64) (float64, bool, bool, float64, error) {
	if err := site.updateMeters(); err != nil {
		return 0, false, false, 0, err
	}

	// allow using PV as estimate for grid power
	if site.gridMeter == nil {
		site.gridPower = totalChargePower - site.pvPower
		site.publish(keys.Grid, types.Measurement{Power: site.gridPower})
	}

	// sitePower adjustment applied for battery priority
	var priorityAdjustment float64

	// ensure safe default for residual power
	residualPower := site.GetResidualPower()
	if len(site.batteryMeters) > 0 && site.battery.Soc < site.prioritySoc && residualPower <= 0 {
		priorityAdjustment += residualPower - 100
		residualPower = 100 // W
	}

	// allow using grid and charge as estimate for pv power
	if site.pvMeters == nil {
		site.pvPower = totalChargePower - site.gridPower + residualPower
		if site.pvPower < 0 {
			site.pvPower = 0
		}
		site.log.DEBUG.Printf("pv power: %.0fW", site.pvPower)
		site.publish(keys.PvPower, site.pvPower)
	}

	// honour battery priority
	batteryPower := site.battery.Power
	excessDCPower := site.excessDCPower

	// handed to loadpoint
	var batteryBuffered, batteryStart bool

	if len(site.batteryMeters) > 0 {
		site.RLock()
		defer site.RUnlock()

		// if battery is charging below prioritySoc give it priority
		if site.battery.Soc < site.prioritySoc && batteryPower < 0 {
			site.log.DEBUG.Printf("battery has priority at soc %.0f%% (< %.0f%%)", site.battery.Soc, site.prioritySoc)
			priorityAdjustment += batteryPower + excessDCPower
			batteryPower = 0
			excessDCPower = 0
		} else {
			// if battery is above bufferSoc allow using it for charging
			batteryBuffered = site.bufferSoc > 0 && site.battery.Soc > site.bufferSoc
			batteryStart = site.bufferStartSoc > 0 && site.battery.Soc >= site.bufferStartSoc
		}
	}

	sitePower := site.gridPower + batteryPower + excessDCPower + residualPower - site.auxPower - flexiblePower

	// handle priority
	var flexStr string
	if flexiblePower > 0 {
		flexStr = fmt.Sprintf(" (including %.0fW prioritized power)", flexiblePower)
	}

	site.log.DEBUG.Printf("site power: %.0fW"+flexStr, sitePower)

	return sitePower, batteryBuffered, batteryStart, priorityAdjustment, nil
}

// updateLoadpoints updates all loadpoints' charge power
func (site *Site) updateLoadpoints(rates api.Rates) float64 {
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		sum float64
	)

	for _, lp := range site.loadpoints {
		wg.Go(func() {
			power := lp.UpdateChargePowerAndCurrents()
			site.prioritizer.UpdateChargePowerFlexibility(lp, rates)

			mu.Lock()
			sum += power
			mu.Unlock()
		})
	}
	wg.Wait()

	return sum
}

// reservedPVPower returns the anticipated surplus claimed by higher-priority PV loadpoints
// that are starting up, so lower-priority loadpoints defer enabling against it (#31194).
func (site *Site) reservedPVPower(lp updater) float64 {
	if lp.GetMode() != api.ModePV {
		return 0
	}

	prio := lp.EffectivePriority()

	var reserved float64
	for _, other := range site.loadpoints {
		if other == lp {
			continue
		}
		if other.EffectivePriority() > prio && other.PvChargeStarting() {
			reserved += other.EffectiveMaxPower()
		}
	}

	if reserved > 0 {
		site.log.DEBUG.Printf("lp %s reserves %.0fW for higher-priority loadpoints starting up", lp.GetTitle(), reserved)
	}

	return reserved
}

func (site *Site) update(lp updater) {
	site.log.DEBUG.Println("----")

	// smart cost and battery mode handling
	consumption, err := site.tariffRates(api.TariffUsagePlanner)
	if err != nil {
		site.log.WARN.Println("planner:", err)
	}

	feedin, err := site.tariffRates(api.TariffUsageFeedIn)
	if err != nil {
		site.log.WARN.Println("feed-in:", err)
	}

	// update loadpoints
	totalChargePower := site.updateLoadpoints(consumption)

	// update all circuits' power and currents
	if site.circuit != nil {
		if err := site.circuit.Update(site.loadpointsAsCircuitDevices()); err != nil {
			site.log.ERROR.Println(err)
		}

		site.publishCircuits()
	}

	if site.hems != nil {
		var wg sync.WaitGroup

		wg.Go(func() {
			if dim := hems.Dimmed(site.hems); dim != nil {
				if err := site.dimMeters(*dim); err != nil {
					site.log.ERROR.Println(err)
				}
			}
		})

		wg.Go(func() {
			if hems.Curtailed(site.hems) != nil {
				if err := site.curtailPV(site.hems.CurtailedPercent()); err != nil {
					site.log.ERROR.Println(err)
				}
			}
		})

		wg.Wait()
	}

	// prioritize if possible
	var flexiblePower float64
	if lp != nil && lp.GetMode() == api.ModePV {
		flexiblePower = site.prioritizer.GetChargePowerFlexibility(lp)
	}

	if sitePower, batteryBuffered, batteryStart, priorityAdjustment, err := site.sitePower(totalChargePower, flexiblePower); err == nil {
		// ignore negative pvPower values as that means it is not an energy source but consumption
		homePower := site.gridPower + max(0, site.pvPower) + site.battery.Power - totalChargePower
		homePower = max(homePower, 0)
		site.publish(keys.HomePower, homePower)

		if homePower > 0 {
			if err := site.collectors[metrics.Home].AddEnergy(nil, nil, homePower); err != nil {
				site.log.ERROR.Printf("persist home consumption: %v", err)
			}
		}

		// add battery charging power to homePower to ignore all consumption which does not occur on loadpoints
		// fix for: https://github.com/evcc-io/evcc/issues/11032
		nonChargePower := homePower + max(0, -site.battery.Power)
		greenShareHome := site.greenShare(0, homePower)
		greenShareLoadpoints := site.greenShare(nonChargePower, nonChargePower+totalChargePower)

		// TODO
		if lp != nil {
			// reserve surplus claimed by higher-priority loadpoints that are starting up (#31194)
			sitePower += site.reservedPVPower(lp)

			// battery boost deliberately drains the battery, hence battery priority
			// below prioritySoc does not apply to the boosting loadpoint (#30541)
			if lp.GetBatteryBoost() != boostDisabled {
				sitePower += priorityAdjustment
			}

			lp.Update(
				sitePower, max(0, site.battery.Power), consumption, feedin, batteryBuffered, batteryStart,
				greenShareLoadpoints, site.effectivePrice(greenShareLoadpoints), site.effectiveCo2(greenShareLoadpoints),
				hems.Dimmed(site.hems),
			)
		}

		site.publishTariffs(greenShareHome, greenShareLoadpoints)

		if telemetry.Enabled() && totalChargePower > standbyPower {
			go telemetry.UpdateChargeProgress(site.log, totalChargePower, greenShareLoadpoints)
		}
	} else {
		site.log.ERROR.Println(err)
	}

	// smart grid charging
	rate, err := consumption.At(time.Now())
	if consumption != nil && err != nil {
		msg := fmt.Sprintf("no matching rate for: %s", time.Now().Format(time.RFC3339))
		if len(consumption) > 0 {
			msg += fmt.Sprintf(", %d consumption rates (%s to %s)", len(consumption),
				consumption[0].Start.Local().Format(time.RFC3339),
				consumption[len(consumption)-1].End.Local().Format(time.RFC3339),
			)
		}

		site.log.WARN.Println("planner:", msg)
	}

	// update battery after reading meters to ensure that (modbus) connection is open
	batteryGridChargeActive := site.batteryGridChargeActive(rate)
	site.publish(keys.BatteryGridChargeActive, batteryGridChargeActive)
	site.updateBatteryMode(batteryGridChargeActive, rate)

	// re-evaluate against the updated loadpoint state
	site.publishSuggestions()

	site.stats.Update(site)
}

// prepare publishes initial values
func (site *Site) prepare() {
	if err := site.restoreSettings(); err != nil {
		site.log.ERROR.Println(err)
	}

	site.publish(keys.SiteTitle, site.Title)

	site.publish(keys.GridConfigured, site.gridMeter != nil)
	site.publish(keys.Grid, api.Meter(nil))
	site.publish(keys.Pv, []api.Meter{})
	site.publish(keys.Aux, []api.Meter{})
	site.publish(keys.Ext, []api.Meter{})
	site.publish(keys.Battery, nil)
	site.publish(keys.PrioritySoc, site.prioritySoc)
	site.publish(keys.BufferSoc, site.bufferSoc)
	site.publish(keys.BufferStartSoc, site.bufferStartSoc)
	site.publish(keys.BatteryMode, site.batteryMode)
	site.publish(keys.BatteryDischargeControl, site.batteryDischargeControl)
	site.publish(keys.BatteryGridDischarge, site.batteryGridDischarge)
	site.publish(keys.SolarAdjusted, site.solarAdjusted)
	site.publish(keys.ResidualPower, site.GetResidualPower())
	site.publish(keys.SmartCostAvailable, site.isDynamicTariff(api.TariffUsagePlanner))
	site.publish(keys.SmartFeedInPriorityAvailable, site.isDynamicTariff(api.TariffUsageFeedIn))

	site.publish(keys.Currency, site.tariffs.Currency)
	if tariff := site.GetTariff(api.TariffUsagePlanner); tariff != nil {
		site.publish(keys.SmartCostType, tariff.Type())
	} else {
		site.publish(keys.SmartCostType, nil)
	}

	site.publishVehicles()
	site.publishTariffs(0, 0)
	vehicle.Publish = site.publishVehicles
	vehicle.ClearPlanLocks = site.clearPlanLocks
}

// Prepare attaches communication channels to site and loadpoints
func (site *Site) Prepare(valueChan chan<- util.Param, pushChan chan<- messenger.Event) {
	site.pushChan = pushChan
	// https://github.com/evcc-io/evcc/issues/11191 prevent deadlock
	// https://github.com/evcc-io/evcc/pull/11675 maintain message order

	// infinite queue with channel semantics
	ch := chanx.NewUnboundedChan[util.Param](context.Background(), 2)

	// use ch.In for writing
	site.valueChan = ch.In

	// use ch.Out for reading
	go func() {
		for p := range ch.Out {
			valueChan <- p
		}
	}()

	site.lpUpdateChan = make(chan *Loadpoint, 1) // 1 capacity to avoid deadlock

	site.prepare()

	lpDevices := config.Loadpoints().Devices()

	for id, lp := range site.loadpoints {
		lpUIChan := make(chan util.Param)
		lpPushChan := make(chan messenger.Event)

		// pipe messages through go func to add id
		go func(id int) {
			for {
				select {
				case param := <-lpUIChan:
					param.Loadpoint = &id
					site.valueChan <- param
				case ev := <-lpPushChan:
					ev.Loadpoint = &id
					pushChan <- ev
				}
			}
		}(id)

		// publish name on the loadpoint's behalf — it doesn't know its own
		if id < len(lpDevices) {
			site.valueChan <- util.Param{Loadpoint: &id, Key: keys.Name, Val: lpDevices[id].Config().Name}
		}

		lp.Prepare(site, lpUIChan, lpPushChan, site.lpUpdateChan)
	}
}

// loopLoadpoints keeps iterating across loadpoints sending the next to the given channel
func (site *Site) loopLoadpoints(next chan<- updater) {
	var logOnce sync.Once

	for {
		if len(site.loadpoints) == 0 {
			logOnce.Do(func() {
				site.log.INFO.Println("no loadpoints configured, running in meter-only mode")
			})
			next <- nil
		} else {
			for _, lp := range site.loadpoints {
				next <- lp
			}
		}
	}
}

// Run is the main control loop. It reacts to trigger events by
// updating measurements and executing control logic.
func (site *Site) Run(stopC chan struct{}, interval time.Duration) {
	if max := 30 * time.Second; interval < max {
		site.log.INFO.Printf("interval <%.0fs can lead to unexpected behavior, see https://docs.evcc.io/docs/reference/configuration/interval", max.Seconds())
	}

	loadpointChan := make(chan updater)
	if site.IsConfigured() {
		go site.loopLoadpoints(loadpointChan)
	}

	site.update(<-loadpointChan) // start immediately

	for tick := time.Tick(interval); ; {
		select {
		case <-tick:
			site.update(<-loadpointChan)
		case lp := <-site.lpUpdateChan:
			site.update(lp)
		case <-stopC:
			return
		}
	}
}
