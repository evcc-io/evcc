package core

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/avast/retry-go/v3"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/db"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/prioritizer"
	"github.com/evcc-io/evcc/push"
	serverdb "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/telemetry"
)

const standbyPower = 10 // consider less than 10W as charger in standby

// Updater abstracts the Loadpoint implementation for testing
type Updater interface {
	loadpoint.API
	Update(availablePower float64, batteryBuffered bool)
}

// meterMeasurement is used as slice element for publishing structured data
type meterMeasurement struct {
	Power float64 `json:"power"`
}

// batteryMeasurement is used as slice element for publishing structured data
type batteryMeasurement struct {
	Power    float64 `json:"power"`
	Soc      float64 `json:"soc"`
	Capacity float64 `json:"capacity"`
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	uiChan       chan<- util.Param // client push messages
	lpUpdateChan chan *Loadpoint

	*Health

	sync.Mutex
	log *util.Logger

	// configuration
	Title                             string       `mapstructure:"title"`         // UI title
	Voltage                           float64      `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower                     float64      `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Meters                            MetersConfig // Meter references
	PrioritySoc                       float64      `mapstructure:"prioritySoc"`                       // prefer battery up to this Soc
	BufferSoc                         float64      `mapstructure:"bufferSoc"`                         // ignore battery above this Soc
	MaxGridSupplyWhileBatteryCharging float64      `mapstructure:"maxGridSupplyWhileBatteryCharging"` // ignore battery charging if AC consumption is above this value

	// meters
	gridMeter     api.Meter   // Grid usage meter
	pvMeters      []api.Meter // PV generation meters
	batteryMeters []api.Meter // Battery charging meters
	auxMeters     []api.Meter // Auxiliary meters

	tariffs     tariff.Tariffs           // Tariff
	loadpoints  []*Loadpoint             // Loadpoints
	coordinator *coordinator.Coordinator // Vehicles
	prioritizer *prioritizer.Prioritizer // Power budgets
	savings     *Savings                 // Savings

	// cached state
	gridPower    float64 // Grid power
	pvPower      float64 // PV power
	batteryPower float64 // Battery charge power
	batterySoc   float64 // Battery soc

	publishCache map[string]any // store last published values to avoid unnecessary republishing
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef      string   `mapstructure:"grid"`      // Grid usage meter
	PVMetersRef       []string `mapstructure:"pv"`        // PV meter
	PVMetersRef_      []string `mapstructure:"pvs"`       // TODO deprecated
	BatteryMetersRef  []string `mapstructure:"battery"`   // Battery charging meter
	BatteryMetersRef_ []string `mapstructure:"batteries"` // TODO deprecated
	AuxMetersRef      []string `mapstructure:"aux"`       // Auxiliary meters
}

// NewSiteFromConfig creates a new site
func NewSiteFromConfig(
	log *util.Logger,
	cp configProvider,
	other map[string]interface{},
	loadpoints []*Loadpoint,
	vehicles []api.Vehicle,
	tariffs tariff.Tariffs,
) (*Site, error) {
	site := NewSite()
	if err := util.DecodeOther(other, site); err != nil {
		return nil, err
	}

	Voltage = site.Voltage
	site.loadpoints = loadpoints
	site.tariffs = tariffs
	site.coordinator = coordinator.New(log, vehicles)
	site.prioritizer = prioritizer.New()
	site.savings = NewSavings(tariffs)

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

		if serverdb.Instance != nil {
			var err error
			if lp.db, err = db.New(lp.Title()); err != nil {
				return nil, err
			}

			// NOTE: this requires stopSession to respect async access
			shutdown.Register(lp.stopSession)
		}
	}

	// grid meter
	if site.Meters.GridMeterRef != "" {
		var err error
		if site.gridMeter, err = cp.Meter(site.Meters.GridMeterRef); err != nil {
			return nil, err
		}
	}

	// multiple pv
	for _, ref := range append(site.Meters.PVMetersRef, site.Meters.PVMetersRef_...) {
		pv, err := cp.Meter(ref)
		if err != nil {
			return nil, err
		}
		site.pvMeters = append(site.pvMeters, pv)
	}

	// multiple batteries
	for _, ref := range append(site.Meters.BatteryMetersRef, site.Meters.BatteryMetersRef_...) {
		battery, err := cp.Meter(ref)
		if err != nil {
			return nil, err
		}
		site.batteryMeters = append(site.batteryMeters, battery)
	}

	// auxiliary meters
	for _, ref := range site.Meters.AuxMetersRef {
		meter, err := cp.Meter(ref)
		if err != nil {
			return nil, err
		}
		site.auxMeters = append(site.auxMeters, meter)
	}

	// configure meter from references
	if site.gridMeter == nil && len(site.pvMeters) == 0 {
		return nil, errors.New("missing either grid or pv meter")
	}

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

// Loadpoints returns the array of associated loadpoints
func (site *Site) Loadpoints() []loadpoint.API {
	res := make([]loadpoint.API, len(site.loadpoints))
	for id, lp := range site.loadpoints {
		res[id] = lp
	}
	return res
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
	if vehicles := site.GetVehicles(); len(vehicles) > 1 {
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

	if vehicles := site.GetVehicles(); len(vehicles) > 0 {
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

// updateMeter updates and publishes single meter
func (site *Site) updateMeters() error {
	retryMeter := func(name string, meter api.Meter, power *float64) error {
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

	if len(site.pvMeters) > 0 {
		site.pvPower = 0
		mm := make([]meterMeasurement, len(site.pvMeters))

		for i, meter := range site.pvMeters {
			var power float64
			err := retry.Do(site.updateMeter(meter, &power), retryOptions...)

			mm[i] = meterMeasurement{Power: power}

			if err == nil {
				// ignore negative values which represent self-consumption
				site.pvPower += math.Max(0, power)
				if power < -500 {
					site.log.WARN.Printf("pv %d power: %.0fW is negative - check configuration if sign is correct", i+1, power)
				}
			} else {
				err = fmt.Errorf("pv %d power: %v", i+1, err)
				site.log.ERROR.Println(err)
			}
		}

		site.log.DEBUG.Printf("pv power: %.0fW", site.pvPower)
		site.publish("pvPower", site.pvPower)

		site.publish("pv", mm)
	}

	if len(site.batteryMeters) > 0 {
		var totalCapacity float64
		site.batteryPower = 0
		site.batterySoc = 0

		mm := make([]batteryMeasurement, len(site.batteryMeters))

		for i, meter := range site.batteryMeters {
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

			var capacity float64
			soc, err := meter.(api.Battery).Soc()

			if err == nil {
				// weigh soc by capacity and accumulate total capacity
				weighedSoc := soc
				if m, ok := meter.(api.BatteryCapacity); ok {
					capacity = m.Capacity()
					totalCapacity += capacity
					weighedSoc *= capacity
				}

				site.batterySoc += weighedSoc
				if len(site.batteryMeters) > 1 {
					site.log.DEBUG.Printf("battery %d soc: %.0f%%", i+1, soc)
				}
			} else {
				site.log.ERROR.Printf("battery %d soc: %v", i+1, err)
			}

			mm[i] = batteryMeasurement{
				Power:    power,
				Soc:      soc,
				Capacity: capacity,
			}
		}

		site.publish("batteryCapacity", math.Round(totalCapacity))

		// convert weighed socs to total soc
		if totalCapacity == 0 {
			totalCapacity = float64(len(site.batteryMeters))
		}
		site.batterySoc /= totalCapacity

		site.log.DEBUG.Printf("battery soc: %.0f%%", math.Round(site.batterySoc))
		site.publish("batterySoc", math.Round(site.batterySoc))

		site.log.DEBUG.Printf("battery power: %.0fW", site.batteryPower)
		site.publish("batteryPower", site.batteryPower)

		site.publish("battery", mm)
	}

	err := retryMeter("grid", site.gridMeter, &site.gridPower)

	// powers
	var p1, p2, p3 float64
	if phaseMeter, ok := site.gridMeter.(api.PhasePowers); err == nil && ok {
		p1, p2, p3, err = phaseMeter.Powers()
		if err == nil {
			phases := []float64{p1, p2, p3}
			site.log.DEBUG.Printf("grid powers: %.0fW", phases)
			site.publish("gridPowers", phases)
		} else {
			err = fmt.Errorf("grid powers: %w", err)
		}
	}

	// currents
	if phaseMeter, ok := site.gridMeter.(api.PhaseCurrents); err == nil && ok {
		var i1, i2, i3 float64
		i1, i2, i3, err = phaseMeter.Currents()
		if err == nil {
			phases := []float64{util.SignFromPower(i1, p1), util.SignFromPower(i2, p2), util.SignFromPower(i3, p3)}
			site.log.DEBUG.Printf("grid currents: %.3gA", phases)
			site.publish("gridCurrents", phases)
		} else {
			err = fmt.Errorf("grid currents: %w", err)
		}
	}

	// energy
	if energyMeter, ok := site.gridMeter.(api.MeterEnergy); err == nil && ok {
		val, err := energyMeter.TotalEnergy()
		if err == nil {
			site.publish("gridEnergy", val)
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
func (site *Site) sitePower(totalChargePower, flexiblePower float64) (float64, bool, error) {
	if err := site.updateMeters(); err != nil {
		return 0, false, err
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
		site.publish("pvPower", site.pvPower)
	}

	// honour battery priority
	batteryPower := site.batteryPower

	// handed to loadpoint
	var batteryBuffered bool

	if len(site.batteryMeters) > 0 {
		site.Lock()
		defer site.Unlock()

		// if battery is charging below prioritySoc give it priority
		if site.batterySoc < site.PrioritySoc && batteryPower < 0 {
			site.log.DEBUG.Printf("giving priority to battery charging at soc: %.0f%%", site.batterySoc)
			batteryPower = 0
		} else {
			// if battery is above bufferSoc allow using it for charging
			batteryBuffered = site.BufferSoc > 0 && site.batterySoc > site.BufferSoc
		}
	}

	sitePower := sitePower(site.log, site.MaxGridSupplyWhileBatteryCharging, site.gridPower, batteryPower, site.ResidualPower)

	// deduct smart loads
	var auxPower float64
	for i, meter := range site.auxMeters {
		if power, err := meter.CurrentPower(); err != nil {
			site.log.ERROR.Printf("aux meter %d: %v", i, err)
		} else {
			auxPower += power
			site.log.DEBUG.Printf("aux power %d: %.0fW", i, power)
		}
	}
	site.publish("auxPower", auxPower)
	sitePower -= auxPower

	// handle priority
	if flexiblePower > 0 {
		site.log.DEBUG.Printf("giving loadpoint priority for additional: %.0fW", flexiblePower)
		sitePower -= flexiblePower
	}

	site.log.DEBUG.Printf("site power: %.0fW", sitePower)

	return sitePower, batteryBuffered, nil
}

func (site *Site) greenShare() float64 {
	batteryDischarge := math.Max(0, site.batteryPower)
	batteryCharge := -math.Min(0, site.batteryPower)
	pvConsumption := math.Min(site.pvPower, site.pvPower+site.gridPower-batteryCharge)

	gridImport := math.Max(0, site.gridPower)
	selfConsumption := math.Max(0, batteryDischarge+pvConsumption+batteryCharge)

	share := selfConsumption / (gridImport + selfConsumption)

	if math.IsNaN(share) {
		return 0
	}

	return share
}

// effectivePrice calculates the real energy price based on self-produced and grid-imported energy.
func (s *Site) effectivePrice(greenShare float64) (float64, error) {
	if grid, err := s.tariffs.CurrentGridPrice(); err == nil {
		feedin, err := s.tariffs.CurrentFeedInPrice()
		if err != nil {
			feedin = 0
		}
		return grid*(1-greenShare) + feedin*greenShare, nil
	}
	return 0, api.ErrNotAvailable
}

// effectiveCo2 calculates the amount of emitted co2 based on self-produced and grid-imported energy.
func (s *Site) effectiveCo2(greenShare float64) (float64, error) {
	if co2, err := s.tariffs.CurrentCo2(); err == nil {
		return co2 * (1 - greenShare), nil
	}
	return 0, api.ErrNotAvailable
}

func (s *Site) publishTariffs() {
	greenShare := s.greenShare()

	s.publish("greenShare", greenShare)

	if gridPrice, err := s.tariffs.CurrentGridPrice(); err == nil {
		s.publishDelta("tariffGrid", gridPrice)
	}
	if feedInPrice, err := s.tariffs.CurrentFeedInPrice(); err == nil {
		s.publishDelta("tariffFeedIn", feedInPrice)
	}
	if co2, err := s.tariffs.CurrentCo2(); err == nil {
		s.publishDelta("tariffCo2", co2)
	}
	if price, err := s.effectivePrice(greenShare); err == nil {
		s.publish("tariffEffectivePrice", price)
	}
	if co2, err := s.effectiveCo2(greenShare); err == nil {
		s.publish("tariffEffectiveCo2", co2)
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

	if sitePower, batteryBuffered, err := site.sitePower(totalChargePower, flexiblePower); err == nil {
		lp.Update(sitePower, batteryBuffered)

		// ignore negative pvPower values as that means it is not an energy source but consumption
		homePower := site.gridPower + math.Max(0, site.pvPower) + site.batteryPower - totalChargePower
		homePower = math.Max(homePower, 0)
		site.publish("homePower", homePower)

		site.Health.Update()
	}

	site.publishTariffs()
	greenShare := site.greenShare()

	// TODO: use energy instead of current power for better results
	deltaCharged := site.savings.Update(site, greenShare, totalChargePower)
	if telemetry.Enabled() && totalChargePower > standbyPower {
		go telemetry.UpdateChargeProgress(site.log, totalChargePower, deltaCharged, greenShare)
	}
}

// prepare publishes initial values
func (site *Site) prepare() {
	site.publish("siteTitle", site.Title)

	site.publish("gridConfigured", site.gridMeter != nil)
	site.publish("pvConfigured", len(site.pvMeters) > 0)
	site.publish("batteryConfigured", len(site.batteryMeters) > 0)
	site.publish("bufferSoc", site.BufferSoc)
	site.publish("prioritySoc", site.PrioritySoc)
	site.publish("residualPower", site.ResidualPower)

	site.publish("currency", site.tariffs.Currency.String())
	site.publish("savingsSince", site.savings.Since())

	site.publish("vehicles", vehicleTitles(site.GetVehicles()))
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
