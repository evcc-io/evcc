package core

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/prioritizer"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/telemetry"
)

const standbyPower = 10 // consider less than 10W as charger in standby

// Updater abstracts the Loadpoint implementation for testing
type Updater interface {
	loadpoint.API
	Update(availablePower float64, autoCharge, batteryBuffered, batteryStart bool, greenShare float64, effectivePrice, effectiveCo2 *float64)
}

// meterMeasurement is used as slice element for publishing structured data
type meterMeasurement struct {
	Power  float64 `json:"power"`
	Energy float64 `json:"energy,omitempty"`
}

// batteryMeasurement is used as slice element for publishing structured data
type batteryMeasurement struct {
	Power        float64 `json:"power"`
	Energy       float64 `json:"energy,omitempty"`
	Soc          float64 `json:"soc,omitempty"`
	Capacity     float64 `json:"capacity,omitempty"`
	Controllable bool    `json:"controllable"`
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	uiChan       chan<- util.Param // client push messages
	lpUpdateChan chan *Loadpoint

	*Health

	sync.RWMutex
	log *util.Logger

	// configuration
	Title                             string       `mapstructure:"title"`         // UI title
	Voltage                           float64      `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower                     float64      `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Meters                            MetersConfig // Meter references
	MaxGridSupplyWhileBatteryCharging float64      `mapstructure:"maxGridSupplyWhileBatteryCharging"` // ignore battery charging if AC consumption is above this value

	// meters
	gridMeter     api.Meter   // Grid usage meter
	pvMeters      []api.Meter // PV generation meters
	batteryMeters []api.Meter // Battery charging meters
	auxMeters     []api.Meter // Auxiliary meters

	// cost settings
	smartCostLimit float64 // always charge if cost is below this value

	// battery settings
	prioritySoc             float64 // prefer battery up to this Soc
	bufferSoc               float64 // continue charging on battery above this Soc
	bufferStartSoc          float64 // start charging on battery above this Soc
	batteryDischargeControl bool    // prevent battery discharge for fast and planned charging

	tariffs     tariff.Tariffs           // Tariff
	loadpoints  []*Loadpoint             // Loadpoints
	coordinator *coordinator.Coordinator // Vehicles
	prioritizer *prioritizer.Prioritizer // Power budgets
	stats       *Stats                   // Stats

	// cached state
	gridPower    float64         // Grid power
	pvPower      float64         // PV power
	batteryPower float64         // Battery charge power
	batterySoc   float64         // Battery soc
	batteryMode  api.BatteryMode // Battery mode

	publishCache map[string]any // store last published values to avoid unnecessary republishing
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef     string   `mapstructure:"grid"`    // Grid usage meter
	PVMetersRef      []string `mapstructure:"pv"`      // PV meter
	BatteryMetersRef []string `mapstructure:"battery"` // Battery charging meter
	AuxMetersRef     []string `mapstructure:"aux"`     // Auxiliary meters
}

// NewSiteFromConfig creates a new site
func NewSiteFromConfig(
	log *util.Logger,
	other map[string]interface{},
	loadpoints []*Loadpoint,
	tariffs tariff.Tariffs,
) (*Site, error) {
	site := NewSite()
	if err := util.DecodeOther(other, site); err != nil {
		return nil, err
	}

	Voltage = site.Voltage
	site.loadpoints = loadpoints
	site.tariffs = tariffs

	handler := config.Vehicles()
	site.coordinator = coordinator.New(log, config.Instances(handler.Devices()))
	handler.Subscribe(site.updateVehicles)

	site.prioritizer = prioritizer.New(log)
	site.stats = NewStats()

	// upload telemetry on shutdown
	if telemetry.Enabled() {
		shutdown.Register(func() {
			telemetry.Persist(log)
		})
	}

	tariff := site.GetTariff(PlannerTariff)

	// give loadpoints access to vehicles and database
	for _, lp := range loadpoints {
		lp.coordinator = coordinator.NewAdapter(lp, site.coordinator)
		lp.planner = planner.New(lp.log, tariff)

		if db.Instance != nil {
			var err error
			if lp.db, err = session.NewStore(lp.Title(), db.Instance); err != nil {
				return nil, err
			}
			// Fix any dangling history
			if err := lp.db.ClosePendingSessionsInHistory(lp.chargeMeterTotal()); err != nil {
				return nil, err
			}

			// NOTE: this requires stopSession to respect async access
			shutdown.Register(lp.stopSession)
		}
	}

	// grid meter
	if site.Meters.GridMeterRef != "" {
		dev, err := config.Meters().ByName(site.Meters.GridMeterRef)
		if err != nil {
			return nil, err
		}
		site.gridMeter = dev.Instance()
	}

	// multiple pv
	for _, ref := range site.Meters.PVMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return nil, err
		}
		site.pvMeters = append(site.pvMeters, dev.Instance())
	}

	// multiple batteries
	for _, ref := range site.Meters.BatteryMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return nil, err
		}
		site.batteryMeters = append(site.batteryMeters, dev.Instance())
	}

	if len(site.batteryMeters) > 0 && site.ResidualPower <= 0 {
		site.log.WARN.Println("battery configured but residualPower is missing or <= 0 (add residualPower: 100 to site), see https://docs.evcc.io/en/docs/reference/configuration/site#residualpower")
	}

	// auxiliary meters
	for _, ref := range site.Meters.AuxMetersRef {
		dev, err := config.Meters().ByName(ref)
		if err != nil {
			return nil, err
		}
		site.auxMeters = append(site.auxMeters, dev.Instance())
	}

	// configure meter from references
	if site.gridMeter == nil && len(site.pvMeters) == 0 {
		return nil, errors.New("missing either grid or pv meter")
	}

	// revert battery mode on shutdown
	shutdown.Register(func() {
		if mode := site.GetBatteryMode(); mode != api.BatteryUnknown && mode != api.BatteryNormal {
			if err := site.updateBatteryMode(api.BatteryNormal); err != nil {
				site.log.ERROR.Println("battery mode:", err)
			}
		}
	})

	return site, nil
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	lp := &Site{
		log:          util.NewLogger("site"),
		publishCache: make(map[string]any),
		Voltage:      230, // V
	}

	return lp
}

// restoreSettings restores site settings
func (site *Site) restoreSettings() error {
	if v, err := settings.Float(keys.BufferSoc); err == nil {
		if err := site.SetBufferSoc(v); err != nil {
			return err
		}
	}
	if v, err := settings.Float(keys.BufferStartSoc); err == nil {
		if err := site.SetBufferStartSoc(v); err != nil {
			return err
		}
	}
	if v, err := settings.Float(keys.SmartCostLimit); err == nil {
		if err := site.SetSmartCostLimit(v); err != nil {
			return err
		}
	}
	if v, err := settings.Float(keys.PrioritySoc); err == nil {
		if err := site.SetPrioritySoc(v); err != nil {
			return err
		}
	}
	if v, err := settings.Bool(keys.BatteryDischargeControl); err == nil {
		site.batteryDischargeControl = v
	}
	return nil
}

func meterCapabilities(name string, meter interface{}) string {
	_, power := meter.(api.Meter)
	_, energy := meter.(api.MeterEnergy)
	_, currents := meter.(api.PhaseCurrents)

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
			if _, ok := v.(api.ChargeState); !ok {
				site.log.WARN.Printf("vehicle '%s' does not support automatic detection", v.Title())
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
			site.log.INFO.Println(meterCapabilities(fmt.Sprintf("pv %d", i+1), pv))
		}
	}

	if len(site.batteryMeters) > 0 {
		for i, battery := range site.batteryMeters {
			_, ok := battery.(api.Battery)
			_, hasCapacity := battery.(api.BatteryCapacity)

			site.log.INFO.Println(
				meterCapabilities(fmt.Sprintf("battery %d", i+1), battery),
				fmt.Sprintf("soc %s capacity %s", presence[ok], presence[hasCapacity]),
			)
		}
	}

	if vehicles := site.Vehicles().Instances(); len(vehicles) > 0 {
		site.log.INFO.Println("  vehicles:")

		for i, v := range vehicles {
			_, rng := v.(api.VehicleRange)
			_, finish := v.(api.VehicleFinishTimer)
			_, status := v.(api.ChargeState)
			_, climate := v.(api.VehicleClimater)
			_, wakeup := v.(api.Resurrector)
			site.log.INFO.Printf("    vehicle %d: range %s finish %s status %s climate %s wakeup %s",
				i+1, presence[rng], presence[finish], presence[status], presence[climate], presence[wakeup],
			)
		}
	}

	for i, lp := range site.loadpoints {
		lp.log.INFO.Printf("loadpoint %d:", i+1)
		lp.log.INFO.Printf("  mode:        %s", lp.GetMode())

		_, power := lp.charger.(api.Meter)
		_, energy := lp.charger.(api.MeterEnergy)
		_, currents := lp.charger.(api.PhaseCurrents)
		_, phases := lp.charger.(api.PhaseSwitcher)
		_, wakeup := lp.charger.(api.Resurrector)

		lp.log.INFO.Printf("  charger:     power %s energy %s currents %s phases %s wakeup %s",
			presence[power],
			presence[energy],
			presence[currents],
			presence[phases],
			presence[wakeup],
		)

		lp.log.INFO.Printf("  meters:      charge %s", presence[lp.HasChargeMeter()])

		if lp.HasChargeMeter() {
			lp.log.INFO.Printf(meterCapabilities("charge", lp.chargeMeter))
		}
	}
}

// publish sends values to UI and databases
func (site *Site) publish(key string, val interface{}) {
	// test helper
	if site.uiChan == nil {
		return
	}

	if s, ok := val.(fmt.Stringer); ok {
		val = s.String()
	}

	site.uiChan <- util.Param{
		Key: key,
		Val: val,
	}
}

// publishDelta deduplicates messages before publishing
func (site *Site) publishDelta(key string, val interface{}) {
	if v, ok := site.publishCache[key]; ok && v == val {
		return
	}

	site.publishCache[key] = val
	site.publish(key, val)
}

// updateMeter updates and publishes single meter
func (site *Site) updateMeter(meter api.Meter, power *float64) func() error {
	return func() error {
		value, err := meter.CurrentPower()
		if err == nil {
			*power = value // update value if no error
		}

		return err
	}
}

// retryMeter retries meter update
func (site *Site) retryMeter(name string, meter api.Meter, power *float64) error {
	if meter == nil {
		return nil
	}

	err := retry.Do(site.updateMeter(meter, power), retryOptions...)

	if err == nil {
		site.log.DEBUG.Printf("%s power: %.0fW", name, *power)
		site.publish(name+"Power", *power)
	} else {
		err = fmt.Errorf("%s meter: %v", name, err)
		site.log.ERROR.Println(err)
	}

	return err
}

// updateMeter updates and publishes single meter
func (site *Site) updateMeters() error {
	if len(site.pvMeters) > 0 {
		var totalEnergy float64

		site.pvPower = 0

		mm := make([]meterMeasurement, len(site.pvMeters))

		for i, meter := range site.pvMeters {
			// pv power
			var power float64
			err := retry.Do(site.updateMeter(meter, &power), retryOptions...)

			if err == nil {
				// ignore negative values which represent self-consumption
				site.pvPower += max(0, power)
				if power < -500 {
					site.log.WARN.Printf("pv %d power: %.0fW is negative - check configuration if sign is correct", i+1, power)
				}
			} else {
				err = fmt.Errorf("pv %d power: %v", i+1, err)
				site.log.ERROR.Println(err)
			}

			// pv energy (production)
			var energy float64
			if m, ok := meter.(api.MeterEnergy); err == nil && ok {
				energy, err = m.TotalEnergy()
				if err == nil {
					totalEnergy += energy
				} else {
					site.log.ERROR.Printf("pv %d energy: %v", i+1, err)
				}
			}

			mm[i] = meterMeasurement{
				Power:  power,
				Energy: energy,
			}
		}

		site.log.DEBUG.Printf("pv power: %.0fW", site.pvPower)
		site.publish(keys.PvPower, site.pvPower)

		site.publish(keys.PvEnergy, totalEnergy)

		site.publish(keys.Pv, mm)
	}

	if len(site.batteryMeters) > 0 {
		var totalCapacity float64
		var totalEnergy float64

		site.batteryPower = 0
		site.batterySoc = 0

		mm := make([]batteryMeasurement, len(site.batteryMeters))

		for i, meter := range site.batteryMeters {
			// battery power
			var power float64

			// NOTE battery errors are logged but ignored as we don't consider them relevant
			err := retry.Do(site.updateMeter(meter, &power), retryOptions...)

			if err == nil {
				site.batteryPower += power
				if len(site.batteryMeters) > 1 {
					site.log.DEBUG.Printf("battery %d power: %.0fW", i+1, power)
				}
			} else {
				site.log.ERROR.Printf("battery %d power: %v", i+1, err)
			}

			// battery energy (discharge)
			var energy float64
			if m, ok := meter.(api.MeterEnergy); err == nil && ok {
				energy, err = m.TotalEnergy()
				if err == nil {
					totalEnergy += energy
				} else {
					site.log.ERROR.Printf("battery %d energy: %v", i+1, err)
				}
			}

			// battery soc and capacity
			var batSoc, capacity float64
			if meter, ok := meter.(api.Battery); ok {
				batSoc, err = soc.Guard(meter.Soc())

				if err == nil {
					// weigh soc by capacity and accumulate total capacity
					weighedSoc := batSoc
					if m, ok := meter.(api.BatteryCapacity); ok {
						capacity = m.Capacity()
						totalCapacity += capacity
						weighedSoc *= capacity
					}

					site.batterySoc += weighedSoc
					if len(site.batteryMeters) > 1 {
						site.log.DEBUG.Printf("battery %d soc: %.0f%%", i+1, batSoc)
					}
				} else {
					site.log.ERROR.Printf("battery %d soc: %v", i+1, err)
				}
			}

			_, controllable := meter.(api.BatteryController)

			mm[i] = batteryMeasurement{
				Power:        power,
				Energy:       energy,
				Soc:          batSoc,
				Capacity:     capacity,
				Controllable: controllable,
			}
		}

		site.publish(keys.BatteryCapacity, totalCapacity)

		// convert weighed socs to total soc
		if totalCapacity == 0 {
			totalCapacity = float64(len(site.batteryMeters))
		}
		site.batterySoc /= totalCapacity

		site.log.DEBUG.Printf("battery soc: %.0f%%", math.Round(site.batterySoc))
		site.publish(keys.BatterySoc, site.batterySoc)

		site.log.DEBUG.Printf("battery power: %.0fW", site.batteryPower)
		site.publish(keys.BatteryPower, site.batteryPower)

		site.publish(keys.BatteryEnergy, totalEnergy)

		site.publish(keys.Battery, mm)
	}

	// grid power
	err := site.retryMeter("grid", site.gridMeter, &site.gridPower)

	// grid phase powers
	var p1, p2, p3 float64
	if phaseMeter, ok := site.gridMeter.(api.PhasePowers); err == nil && ok {
		p1, p2, p3, err = phaseMeter.Powers()
		if err == nil {
			phases := []float64{p1, p2, p3}
			site.log.DEBUG.Printf("grid powers: %.0fW", phases)
			site.publish(keys.GridPowers, phases)
		} else {
			err = fmt.Errorf("grid powers: %w", err)
		}
	}

	// grid phase currents (signed)
	if phaseMeter, ok := site.gridMeter.(api.PhaseCurrents); err == nil && ok {
		var i1, i2, i3 float64
		i1, i2, i3, err = phaseMeter.Currents()
		if err == nil {
			phases := []float64{util.SignFromPower(i1, p1), util.SignFromPower(i2, p2), util.SignFromPower(i3, p3)}
			site.log.DEBUG.Printf("grid currents: %.3gA", phases)
			site.publish(keys.GridCurrents, phases)
		} else {
			err = fmt.Errorf("grid currents: %w", err)
		}
	}

	// grid energy (import)
	if energyMeter, ok := site.gridMeter.(api.MeterEnergy); err == nil && ok {
		var f float64
		f, err = energyMeter.TotalEnergy()
		if err == nil {
			site.publish(keys.GridEnergy, f)
		} else {
			site.log.ERROR.Printf("grid energy: %v", err)
		}
	}

	return err
}

// sitePower returns
//   - the net power exported by the site minus a residual margin
//     (negative values mean grid: export, battery: charging
//   - if battery buffer can be used for charging
func (site *Site) sitePower(totalChargePower, flexiblePower float64) (float64, bool, bool, error) {
	if err := site.updateMeters(); err != nil {
		return 0, false, false, err
	}

	// allow using PV as estimate for grid power
	if site.gridMeter == nil {
		site.gridPower = totalChargePower - site.pvPower
	}

	// allow using grid and charge as estimate for pv power
	if site.pvMeters == nil {
		site.pvPower = totalChargePower - site.gridPower + site.ResidualPower
		if site.pvPower < 0 {
			site.pvPower = 0
		}
		site.log.DEBUG.Printf("pv power: %.0fW", site.pvPower)
		site.publish(keys.PvPower, site.pvPower)
	}

	// honour battery priority
	batteryPower := site.batteryPower

	// handed to loadpoint
	var batteryBuffered, batteryStart bool

	if len(site.batteryMeters) > 0 {
		site.RLock()
		defer site.RUnlock()

		// if battery is charging below prioritySoc give it priority
		if site.batterySoc < site.prioritySoc && batteryPower < 0 {
			site.log.DEBUG.Printf("battery has priority at soc %.0f%% (< %.0f%%)", site.batterySoc, site.prioritySoc)
			batteryPower = 0
		} else {
			// if battery is above bufferSoc allow using it for charging
			batteryBuffered = site.bufferSoc > 0 && site.batterySoc > site.bufferSoc
			batteryStart = site.bufferStartSoc > 0 && site.batterySoc > site.bufferStartSoc
		}
	}

	sitePower := sitePower(site.log, site.MaxGridSupplyWhileBatteryCharging, site.gridPower, batteryPower, site.ResidualPower)

	// deduct smart loads
	if len(site.auxMeters) > 0 {
		var auxPower float64
		mm := make([]meterMeasurement, len(site.auxMeters))

		for i, meter := range site.auxMeters {
			if power, err := meter.CurrentPower(); err == nil {
				auxPower += power
				mm[i].Power = power
				site.log.DEBUG.Printf("aux power %d: %.0fW", i+1, power)
			} else {
				site.log.ERROR.Printf("aux meter %d: %v", i+1, err)
			}
		}

		sitePower -= auxPower

		site.log.DEBUG.Printf("aux power: %.0fW", auxPower)
		site.publish(keys.AuxPower, auxPower)

		site.publish(keys.Aux, mm)
	}

	// handle priority
	if flexiblePower > 0 {
		site.log.DEBUG.Printf("giving loadpoint priority for additional: %.0fW", flexiblePower)
		sitePower -= flexiblePower
	}

	site.log.DEBUG.Printf("site power: %.0fW", sitePower)

	return sitePower, batteryBuffered, batteryStart, nil
}

// greenShare returns
//   - the current green share, calculated for the part of the consumption between powerFrom and powerTo
//     the consumption below powerFrom will get the available green power first
func (site *Site) greenShare(powerFrom float64, powerTo float64) float64 {
	greenPower := math.Max(0, site.pvPower) + math.Max(0, site.batteryPower)
	greenPowerAvailable := math.Max(0, greenPower-powerFrom)

	power := powerTo - powerFrom
	share := math.Min(greenPowerAvailable, power) / power

	if math.IsNaN(share) {
		if greenPowerAvailable > 0 {
			share = 1
		} else {
			share = 0
		}
	}

	return share
}

// effectivePrice calculates the real energy price based on self-produced and grid-imported energy.
func (site *Site) effectivePrice(greenShare float64) *float64 {
	if grid, err := site.tariffs.CurrentGridPrice(); err == nil {
		feedin, err := site.tariffs.CurrentFeedInPrice()
		if err != nil {
			feedin = 0
		}
		effPrice := grid*(1-greenShare) + feedin*greenShare
		return &effPrice
	}
	return nil
}

// effectiveCo2 calculates the amount of emitted co2 based on self-produced and grid-imported energy.
func (site *Site) effectiveCo2(greenShare float64) *float64 {
	if co2, err := site.tariffs.CurrentCo2(); err == nil {
		effCo2 := co2 * (1 - greenShare)
		return &effCo2
	}
	return nil
}

func (site *Site) publishTariffs(greenShareHome float64, greenShareLoadpoints float64) {
	site.publish(keys.GreenShareHome, greenShareHome)
	site.publish(keys.GreenShareLoadpoints, greenShareLoadpoints)

	if gridPrice, err := site.tariffs.CurrentGridPrice(); err == nil {
		site.publishDelta(keys.TariffGrid, gridPrice)
	}
	if feedInPrice, err := site.tariffs.CurrentFeedInPrice(); err == nil {
		site.publishDelta(keys.TariffFeedIn, feedInPrice)
	}
	if co2, err := site.tariffs.CurrentCo2(); err == nil {
		site.publishDelta(keys.TariffCo2, co2)
	}
	if price := site.effectivePrice(greenShareHome); price != nil {
		site.publish(keys.TariffPriceHome, price)
	}
	if co2 := site.effectiveCo2(greenShareHome); co2 != nil {
		site.publish(keys.TariffCo2Home, co2)
	}
	if price := site.effectivePrice(greenShareLoadpoints); price != nil {
		site.publish(keys.TariffPriceLoadpoints, price)
	}
	if co2 := site.effectiveCo2(greenShareLoadpoints); co2 != nil {
		site.publish(keys.TariffCo2Loadpoints, co2)
	}
}

func (site *Site) update(lp Updater) {
	site.log.DEBUG.Println("----")

	// update all loadpoint's charge power
	var totalChargePower float64
	for _, lp := range site.loadpoints {
		lp.UpdateChargePower()
		totalChargePower += lp.GetChargePower()

		site.prioritizer.UpdateChargePowerFlexibility(lp)
	}

	// prioritize if possible
	var flexiblePower float64
	if lp.GetMode() == api.ModePV {
		flexiblePower = site.prioritizer.GetChargePowerFlexibility(lp)
	}

	var smartCostActive bool
	if tariff := site.GetTariff(PlannerTariff); tariff != nil {
		rates, err := tariff.Rates()

		var rate api.Rate
		if err == nil {
			rate, err = rates.Current(time.Now())
		}

		if err == nil {
			limit := site.GetSmartCostLimit()
			smartCostActive = limit != 0 && rate.Price <= limit
			site.publish(keys.SmartCostActive, smartCostActive)
		} else {
			site.log.ERROR.Println("smartCost:", err)
		}
	}

	if sitePower, batteryBuffered, batteryStart, err := site.sitePower(totalChargePower, flexiblePower); err == nil {
		// ignore negative pvPower values as that means it is not an energy source but consumption
		homePower := site.gridPower + max(0, site.pvPower) + site.batteryPower - totalChargePower
		homePower = max(homePower, 0)
		site.publish(keys.HomePower, homePower)

		greenShareHome := site.greenShare(0, homePower)
		greenShareLoadpoints := site.greenShare(homePower, homePower+totalChargePower)

		lp.Update(sitePower, smartCostActive, batteryBuffered, batteryStart, greenShareLoadpoints, site.effectivePrice(greenShareLoadpoints), site.effectiveCo2(greenShareLoadpoints))

		site.Health.Update()

		site.publishTariffs(greenShareHome, greenShareLoadpoints)

		if telemetry.Enabled() && totalChargePower > standbyPower {
			go telemetry.UpdateChargeProgress(site.log, totalChargePower, greenShareLoadpoints)
		}
	} else {
		site.log.ERROR.Println(err)
	}

	if batMode := site.GetBatteryMode(); site.batteryDischargeControl {
		if mode := site.determineBatteryMode(site.Loadpoints()); mode != batMode {
			if err := site.updateBatteryMode(mode); err != nil {
				site.log.ERROR.Println("battery mode:", err)
			}
		}
	}

	site.stats.Update(site)
}

// prepare publishes initial values
func (site *Site) prepare() {
	site.publish(keys.SiteTitle, site.Title)

	site.publish(keys.GridConfigured, site.gridMeter != nil)
	site.publish(keys.PvConfigured, len(site.pvMeters) > 0)
	site.publish(keys.BatteryConfigured, len(site.batteryMeters) > 0)
	site.publish(keys.BufferSoc, site.bufferSoc)
	site.publish(keys.BufferStartSoc, site.bufferStartSoc)
	site.publish(keys.PrioritySoc, site.prioritySoc)
	site.publish(keys.ResidualPower, site.ResidualPower)
	site.publish(keys.SmartCostLimit, site.smartCostLimit)
	site.publish(keys.SmartCostType, nil)
	site.publish(keys.SmartCostActive, false)
	if tariff := site.GetTariff(PlannerTariff); tariff != nil {
		site.publish(keys.SmartCostType, tariff.Type().String())
	}
	site.publish(keys.Currency, site.tariffs.Currency.String())

	site.publish(keys.BatteryDischargeControl, site.batteryDischargeControl)
	site.publish(keys.BatteryMode, site.batteryMode.String())

	if err := site.restoreSettings(); err != nil {
		site.log.ERROR.Println(err)
	}

	site.publishVehicles()
	vehicle.Publish = site.publishVehicles
}

// Prepare attaches communication channels to site and loadpoints
func (site *Site) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event) {
	site.uiChan = uiChan
	site.lpUpdateChan = make(chan *Loadpoint, 1) // 1 capacity to avoid deadlock

	site.prepare()

	for id, lp := range site.loadpoints {
		lpUIChan := make(chan util.Param)
		lpPushChan := make(chan push.Event)

		// pipe messages through go func to add id
		go func(id int) {
			for {
				select {
				case param := <-lpUIChan:
					param.Loadpoint = &id
					uiChan <- param
				case ev := <-lpPushChan:
					ev.Loadpoint = &id
					pushChan <- ev
				}
			}
		}(id)

		lp.Prepare(lpUIChan, lpPushChan, site.lpUpdateChan)
	}
}

// loopLoadpoints keeps iterating across loadpoints sending the next to the given channel
func (site *Site) loopLoadpoints(next chan<- Updater) {
	for {
		for _, lp := range site.loadpoints {
			next <- lp
		}
	}
}

// Run is the main control loop. It reacts to trigger events by
// updating measurements and executing control logic.
func (site *Site) Run(stopC chan struct{}, interval time.Duration) {
	site.Health = NewHealth(time.Minute + interval)

	if max := 30 * time.Second; interval < max {
		site.log.WARN.Printf("interval <%.0fs can lead to unexpected behavior, see https://docs.evcc.io/docs/reference/configuration/interval", max.Seconds())
	}

	loadpointChan := make(chan Updater)
	go site.loopLoadpoints(loadpointChan)

	ticker := time.NewTicker(interval)
	site.update(<-loadpointChan) // start immediately

	for {
		select {
		case <-ticker.C:
			site.update(<-loadpointChan)
		case lp := <-site.lpUpdateChan:
			site.update(lp)
		case <-stopC:
			return
		}
	}
}
