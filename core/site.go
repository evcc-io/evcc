package core

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/avast/retry-go/v3"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
)

// Updater abstracts the LoadPoint implementation for testing
type Updater interface {
	Update(availablePower float64, cheapRate, batteryBuffered bool)
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	uiChan       chan<- util.Param // client push messages
	lpUpdateChan chan *LoadPoint

	*Health

	sync.Mutex
	log *util.Logger

	// configuration
	Title                             string       `mapstructure:"title"`         // UI title
	Voltage                           float64      `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower                     float64      `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Meters                            MetersConfig // Meter references
	PrioritySoC                       float64      `mapstructure:"prioritySoC"`                       // prefer battery up to this SoC
	BufferSoC                         float64      `mapstructure:"bufferSoC"`                         // ignore battery above this SoC
	MaxGridSupplyWhileBatteryCharging float64      `mapstructure:"maxGridSupplyWhileBatteryCharging"` // ignore battery charging if AC consumption is above this value

	// meters
	gridMeter     api.Meter   // Grid usage meter
	pvMeters      []api.Meter // PV generation meters
	batteryMeters []api.Meter // Battery charging meters

	tariffs    tariff.Tariffs // Tariff
	loadpoints []*LoadPoint   // Loadpoints
	savings    *Savings       // Savings

	// cached state
	gridPower       float64 // Grid power
	pvPower         float64 // PV power
	batteryPower    float64 // Battery charge power
	batteryBuffered bool    // Battery buffer active
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef     string   `mapstructure:"grid"`      // Grid usage meter
	PVMeterRef       string   `mapstructure:"pv"`        // PV meter
	PVMetersRef      []string `mapstructure:"pvs"`       // Multiple PV meters
	BatteryMeterRef  string   `mapstructure:"battery"`   // Battery charging meter
	BatteryMetersRef []string `mapstructure:"batteries"` // Multiple Battery charging meters
}

// NewSiteFromConfig creates a new site
func NewSiteFromConfig(
	log *util.Logger,
	cp configProvider,
	other map[string]interface{},
	loadpoints []*LoadPoint,
	tariffs tariff.Tariffs,
) (*Site, error) {
	site := NewSite()
	if err := util.DecodeOther(other, site); err != nil {
		return nil, err
	}

	Voltage = site.Voltage
	site.loadpoints = loadpoints
	site.tariffs = tariffs
	site.savings = NewSavings(tariffs)

	if site.Meters.GridMeterRef != "" {
		site.gridMeter = cp.Meter(site.Meters.GridMeterRef)
	}

	// multiple pv
	for _, ref := range site.Meters.PVMetersRef {
		pv := cp.Meter(ref)
		site.pvMeters = append(site.pvMeters, pv)
	}

	// single pv
	if site.Meters.PVMeterRef != "" {
		if len(site.pvMeters) > 0 {
			return nil, errors.New("cannot have pv and pvs both")
		}
		pv := cp.Meter(site.Meters.PVMeterRef)
		site.pvMeters = append(site.pvMeters, pv)
	}

	// multiple batteries
	for _, ref := range site.Meters.BatteryMetersRef {
		battery := cp.Meter(ref)
		site.batteryMeters = append(site.batteryMeters, battery)
	}

	// single battery
	if site.Meters.BatteryMeterRef != "" {
		if len(site.batteryMeters) > 0 {
			return nil, errors.New("cannot have battery and batteries both")
		}
		battery := cp.Meter(site.Meters.BatteryMeterRef)
		site.batteryMeters = append(site.batteryMeters, battery)
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
		log:     util.NewLogger("site"),
		Voltage: 230, // V
	}

	return lp
}

// LoadPoints returns the array of associated loadpoints
func (site *Site) LoadPoints() []loadpoint.API {
	res := make([]loadpoint.API, len(site.loadpoints))
	for id, lp := range site.loadpoints {
		res[id] = lp
	}
	return res
}

func meterCapabilities(name string, meter interface{}) string {
	_, power := meter.(api.Meter)
	_, energy := meter.(api.MeterEnergy)
	_, currents := meter.(api.MeterCurrent)

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
			site.log.INFO.Println(
				meterCapabilities(fmt.Sprintf("battery %d", i+1), battery),
				fmt.Sprintf("soc %s", presence[ok]),
			)
		}
	}

	for i, lp := range site.loadpoints {
		lp.log.INFO.Printf("loadpoint %d:", i+1)
		lp.log.INFO.Printf("  mode:        %s", lp.GetMode())

		_, power := lp.charger.(api.Meter)
		_, energy := lp.charger.(api.MeterEnergy)
		_, currents := lp.charger.(api.MeterCurrent)
		_, phases := lp.charger.(api.ChargePhases)
		_, wakeup := lp.charger.(api.Resurrector)

		lp.log.INFO.Printf("  charger:     power %s energy %s currents %s phases %s wakeup %s",
			presence[power],
			presence[energy],
			presence[currents],
			presence[phases],
			presence[wakeup],
		)

		lp.log.INFO.Printf("  meters:      charge %s", presence[lp.HasChargeMeter()])

		lp.publish("chargeConfigured", lp.HasChargeMeter())
		if lp.HasChargeMeter() {
			lp.log.INFO.Printf(meterCapabilities("charge", lp.chargeMeter))
		}

		lp.log.INFO.Printf("  vehicles:    %s", presence[len(lp.vehicles) > 0])

		for i, v := range lp.vehicles {
			_, rng := v.(api.VehicleRange)
			_, finish := v.(api.VehicleFinishTimer)
			_, status := v.(api.ChargeState)
			_, climate := v.(api.VehicleClimater)
			_, wakeup := v.(api.Resurrector)
			lp.log.INFO.Printf("    vehicle %d: range %s finish %s status %s climate %s wakeup %s",
				i+1, presence[rng], presence[finish], presence[status], presence[climate], presence[wakeup],
			)
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

		for id, meter := range site.pvMeters {
			var power float64
			err := retry.Do(site.updateMeter(meter, &power), retryOptions...)

			if err == nil {
				// ignore negative values which represent self-consumption
				site.pvPower += math.Max(0, power)
				if power < -500 {
					site.log.WARN.Printf("pv %d power: %.0fW is negative - check configuration if sign is correct", id, power)
				}
			} else {
				err = fmt.Errorf("pv meter %d: %v", id, err)
				site.log.ERROR.Println(err)
			}
		}

		site.log.DEBUG.Printf("pv power: %.0fW", site.pvPower)
		site.publish("pvPower", site.pvPower)
	}

	if len(site.batteryMeters) > 0 {
		site.batteryPower = 0

		for id, meter := range site.batteryMeters {
			var power float64
			err := retry.Do(site.updateMeter(meter, &power), retryOptions...)

			if err == nil {
				site.batteryPower += power
			} else {
				site.log.ERROR.Println(fmt.Errorf("battery meter %d: %v", id, err))
			}
		}

		site.log.DEBUG.Printf("battery power: %.0fW", site.batteryPower)
		site.publish("batteryPower", site.batteryPower)
	}

	err := retryMeter("grid", site.gridMeter, &site.gridPower)

	// currents
	if phaseMeter, ok := site.gridMeter.(api.MeterCurrent); err == nil && ok {
		i1, i2, i3, err := phaseMeter.Currents()
		if err == nil {
			site.log.DEBUG.Printf("grid currents: %.3gA", []float64{i1, i2, i3})
			site.publish("gridCurrents", []float64{i1, i2, i3})
		} else {
			site.log.ERROR.Println(fmt.Errorf("grid meter currents: %v", err))
		}
	}

	// grid energy
	if energyMeter, ok := site.gridMeter.(api.MeterEnergy); ok {
		val, err := energyMeter.TotalEnergy()
		if err == nil {
			site.publish("gridEnergy", val)
		} else {
			site.log.ERROR.Println(fmt.Errorf("grid meter energy: %v", err))
		}
	}

	return err
}

// sitePower returns the net power exported by the site minus a residual margin.
// negative values mean grid: export, battery: charging
func (site *Site) sitePower(totalChargePower float64) (float64, error) {
	if err := site.updateMeters(); err != nil {
		return 0, err
	}

	// allow using PV as estimate for grid power
	if site.gridMeter == nil {
		site.gridPower = totalChargePower - site.pvPower
	}

	// allow using Grid and charge as estimate for pv power
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

	if len(site.batteryMeters) > 0 {
		var socs float64
		for id, battery := range site.batteryMeters {
			soc, err := battery.(api.Battery).SoC()
			if err != nil {
				err = fmt.Errorf("battery soc %d: %v", id, err)
				site.log.ERROR.Println(err)
			} else {
				site.log.DEBUG.Printf("battery soc %d: %.0f%%", id, soc)
				socs += soc / float64(len(site.batteryMeters))
			}
		}
		site.publish("batterySoC", math.Trunc(socs))

		site.Lock()
		defer site.Unlock()

		// if battery is charging below prioritySoC give it priority
		if socs < site.PrioritySoC && batteryPower < 0 {
			site.log.DEBUG.Printf("giving priority to battery charging at soc: %.0f%%", socs)
			batteryPower = 0
		}

		// if battery is discharging above bufferSoC ignore it
		site.batteryBuffered = batteryPower > 0 && site.BufferSoC > 0 && socs > site.BufferSoC
	}

	sitePower := sitePower(site.log, site.MaxGridSupplyWhileBatteryCharging, site.gridPower, batteryPower, site.ResidualPower)

	site.log.DEBUG.Printf("site power: %.0fW", sitePower)

	return sitePower, nil
}

func (site *Site) update(lp Updater) {
	site.log.DEBUG.Println("----")

	var cheap bool
	var err error
	if site.tariffs.Grid != nil {
		cheap, err = site.tariffs.Grid.IsCheap()
		if err != nil {
			cheap = false
		}
	}

	// update all loadpoint's charge power
	var totalChargePower float64
	for _, lp := range site.loadpoints {
		lp.UpdateChargePower()
		totalChargePower += lp.GetChargePower()
	}

	if sitePower, err := site.sitePower(totalChargePower); err == nil {
		lp.Update(sitePower, cheap, site.batteryBuffered)

		// ignore negative pvPower values as that means it is not an energy source but consumption
		homePower := site.gridPower + math.Max(0, site.pvPower) + site.batteryPower - totalChargePower
		homePower = math.Max(homePower, 0)
		site.publish("homePower", homePower)

		site.Health.Update()
	}

	// update savings
	// TODO: use energy instead of current power for better results
	site.savings.Update(site, site.gridPower, site.pvPower, site.batteryPower, totalChargePower)
}

// prepare publishes initial values
func (site *Site) prepare() {
	site.publish("siteTitle", site.Title)

	site.publish("gridConfigured", site.gridMeter != nil)
	site.publish("pvConfigured", len(site.pvMeters) > 0)
	site.publish("batteryConfigured", len(site.batteryMeters) > 0)
	site.publish("bufferSoC", site.BufferSoC)
	site.publish("prioritySoC", site.PrioritySoC)
	site.publish("residualPower", site.ResidualPower)

	site.publish("currency", site.tariffs.Currency.String())
	site.publish("savingsSince", site.savings.Since().Unix())
}

// Prepare attaches communication channels to site and loadpoints
func (site *Site) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event) {
	site.uiChan = uiChan
	site.lpUpdateChan = make(chan *LoadPoint, 1) // 1 capacity to avoid deadlock

	site.prepare()

	for id, lp := range site.loadpoints {
		lpUIChan := make(chan util.Param)
		lpPushChan := make(chan push.Event)

		// pipe messages through go func to add id
		go func(id int) {
			for {
				select {
				case param := <-lpUIChan:
					param.LoadPoint = &id
					uiChan <- param
				case ev := <-lpPushChan:
					ev.LoadPoint = &id
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
