package core

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	"github.com/avast/retry-go"
	"github.com/pkg/errors"
)

//go:generate mockgen -package mock -destination ../mock/mock_loadpoint.go github.com/andig/evcc/core Updater

// Updater abstracts the LoadPoint implementation for testing
type Updater interface {
	Update(float64)
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	uiChan       chan<- util.Param // client push messages
	lpUpdateChan chan *LoadPoint

	log *util.Logger

	// configuration
	Title         string       `mapstructure:"title"`         // UI title
	Voltage       float64      `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower float64      `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Meters        MetersConfig // Meter references

	// meters
	gridMeter    api.Meter // Grid usage meter
	pvMeter      api.Meter // PV generation meter
	batteryMeter api.Meter // Battery charging meter

	loadpoints []*LoadPoint // Loadpoints

	// cached state
	gridPower    float64 // Grid power
	pvPower      float64 // PV power
	batteryPower float64 // Battery charge power
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef    string `mapstructure:"grid"`    // Grid usage meter reference
	PVMeterRef      string `mapstructure:"pv"`      // PV generation meter reference
	BatteryMeterRef string `mapstructure:"battery"` // Battery charging meter reference
}

// NewSiteFromConfig creates a new site
func NewSiteFromConfig(
	log *util.Logger,
	cp configProvider,
	other map[string]interface{},
	loadpoints []*LoadPoint,
) *Site {
	site := NewSite()
	if err := util.DecodeOther(other, &site); err != nil {
		log.FATAL.Fatal(err)
	}

	Voltage = site.Voltage
	site.loadpoints = loadpoints

	// configure meter from references
	// if site.Meters.PVMeterRef == "" && site.Meters.GridMeterRef == "" {
	// 	site.log.FATAL.Fatal("missing either pv or grid meter")
	// }
	if site.Meters.GridMeterRef == "" {
		site.log.FATAL.Fatal("missing grid meter")
	}
	if site.Meters.GridMeterRef != "" {
		site.gridMeter = cp.Meter(site.Meters.GridMeterRef)
	}
	if site.Meters.PVMeterRef != "" {
		site.pvMeter = cp.Meter(site.Meters.PVMeterRef)
	}
	if site.Meters.BatteryMeterRef != "" {
		site.batteryMeter = cp.Meter(site.Meters.BatteryMeterRef)
	}

	return site
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	lp := &Site{
		log:     util.NewLogger("core"),
		Voltage: 230, // V
	}

	return lp
}

// SiteConfiguration contains the global site configuration
type SiteConfiguration struct {
	Title        string                   `json:"title"`
	GridMeter    bool                     `json:"gridMeter"`
	PVMeter      bool                     `json:"pvMeter"`
	BatteryMeter bool                     `json:"batteryMeter"`
	LoadPoints   []LoadpointConfiguration `json:"loadpoints"`
}

// LoadpointConfiguration is the loadpoint feature structure
type LoadpointConfiguration struct {
	Mode        string `json:"mode"`
	Title       string `json:"title"`
	Phases      int64  `json:"phases"`
	MinCurrent  int64  `json:"minCurrent"`
	MaxCurrent  int64  `json:"maxCurrent"`
	ChargeMeter bool   `json:"chargeMeter"`
	SoC         bool   `json:"soc"`
	SoCCapacity int64  `json:"socCapacity"`
	SoCTitle    string `json:"socTitle"`
	SoCLevels   []int  `json:"socLevels"`
	TargetSoC   int    `json:"targetSoC"`
}

// GetMode Gets loadpoint charge mode
func (site *Site) GetMode() api.ChargeMode {
	return site.loadpoints[0].GetMode()
}

// GetTargetSoC gets loadpoint charge targetSoC
func (site *Site) GetTargetSoC() int {
	return site.loadpoints[0].GetTargetSoC()
}

// SetMode sets loadpoint charge mode
func (site *Site) SetMode(mode api.ChargeMode) {
	site.log.INFO.Printf("set global charge mode: %s", string(mode))
	for _, lp := range site.loadpoints {
		lp.SetMode(mode)
	}
}

// SetTargetSoC sets loadpoint charge targetSoC
func (site *Site) SetTargetSoC(targetSoC int) {
	site.log.INFO.Println("set global target soc:", targetSoC)
	for _, lp := range site.loadpoints {
		lp.SetTargetSoC(targetSoC)
	}
}

func (lp *LoadPoint) hasChargeMeter() bool {
	_, isWrapped := lp.chargeMeter.(*wrapper.ChargeMeter)
	return lp.chargeMeter != nil && !isWrapped
}

// LoadPoints returns the array of associated loadpoints
func (site *Site) LoadPoints() []*LoadPoint {
	return site.loadpoints
}

// Configuration returns meter configuration
func (site *Site) Configuration() SiteConfiguration {
	c := SiteConfiguration{
		Title:        site.Title,
		GridMeter:    site.gridMeter != nil,
		PVMeter:      site.pvMeter != nil,
		BatteryMeter: site.batteryMeter != nil,
	}

	for _, lp := range site.loadpoints {
		lpc := LoadpointConfiguration{
			Mode:        string(lp.GetMode()),
			Title:       lp.Name(),
			Phases:      lp.Phases,
			MinCurrent:  lp.MinCurrent,
			MaxCurrent:  lp.MaxCurrent,
			ChargeMeter: lp.hasChargeMeter(),
		}

		if lp.vehicle != nil {
			lpc.SoC = true
			lpc.SoCCapacity = lp.vehicle.Capacity()
			lpc.SoCTitle = lp.vehicle.Title()
			lpc.SoCLevels = lp.SoC.Levels
			lpc.TargetSoC = lp.TargetSoC
		}

		c.LoadPoints = append(c.LoadPoints, lpc)
	}

	return c
}

// DumpConfig site configuration
func (site *Site) DumpConfig() {
	site.log.INFO.Println("site config:")
	site.log.INFO.Printf("  grid %s", presence[site.gridMeter != nil])
	site.log.INFO.Printf("  pv %s", presence[site.pvMeter != nil])
	site.log.INFO.Printf("  battery %s", presence[site.batteryMeter != nil])

	if site.gridMeter != nil {
		_, power := site.gridMeter.(api.Meter)
		_, energy := site.gridMeter.(api.MeterEnergy)
		_, currents := site.gridMeter.(api.MeterCurrent)

		site.log.INFO.Println("  grid config:")
		site.log.INFO.Printf("    power %s", presence[power])
		site.log.INFO.Printf("    energy %s", presence[energy])
		site.log.INFO.Printf("    currents %s", presence[currents])
	}

	for i, lp := range site.loadpoints {
		lp.log.INFO.Printf("loadpoint %d config:", i+1)

		lp.log.INFO.Printf("  vehicle %s", presence[lp.vehicle != nil])
		lp.log.INFO.Printf("  charge %s", presence[lp.hasChargeMeter()])

		charger := lp.handler.(*ChargerHandler).charger
		_, power := charger.(api.Meter)
		_, currents := charger.(api.MeterCurrent)
		_, energy := charger.(api.ChargeRater)
		_, timer := charger.(api.ChargeTimer)

		lp.log.INFO.Println("  charger config:")
		lp.log.INFO.Printf("    power %s", presence[power])
		lp.log.INFO.Printf("    energy %s", presence[energy])
		lp.log.INFO.Printf("    currents %s", presence[currents])
		lp.log.INFO.Printf("    timer %s", presence[timer])
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
func (site *Site) updateMeter(name string, meter api.Meter, power *float64) error {
	value, err := meter.CurrentPower()
	if err != nil {
		return err
	}

	*power = value // update value if no error

	site.log.DEBUG.Printf("%s power: %.0fW", name, *power)
	site.publish(name+"Power", *power)

	return nil
}

// updateMeter updates and publishes single meter
func (site *Site) updateMeters() error {
	retryMeter := func(s string, m api.Meter, f *float64) error {
		if m == nil {
			return nil
		}

		err := retry.Do(func() error {
			return site.updateMeter(s, m, f)
		}, retryOptions...)

		if err != nil {
			err = errors.Wrapf(err, "updating %s meter", s)
			site.log.ERROR.Println(err)
		}

		return err
	}

	// pv meter is not critical for operation
	_ = retryMeter("pv", site.pvMeter, &site.pvPower)

	err := retryMeter("grid", site.gridMeter, &site.gridPower)
	if err == nil {
		err = retryMeter("battery", site.batteryMeter, &site.batteryPower)
	}

	return err
}

// consumedPower estimates how much power the charger might have consumed given it was the only load
// func (site *Site) consumedPower() float64 {
// 	return consumedPower(lp.pvPower, lp.batteryPower, lp.gridPower)
// }

// sitePower returns the net power exported by the site minus a residual margin.
// negative values mean grid: export, battery: charging
func (site *Site) sitePower() (float64, error) {
	if err := site.updateMeters(); err != nil {
		return 0, err
	}

	sitePower := sitePower(site.gridPower, site.batteryPower, site.ResidualPower)
	site.log.DEBUG.Printf("site power: %.0fW", sitePower)

	return sitePower, nil
}

func (site *Site) update(lp Updater) {
	site.log.DEBUG.Println("----")

	if sitePower, err := site.sitePower(); err == nil {
		lp.Update(sitePower)
	}
}

// Prepare attaches communication channels to site and loadpoints
func (site *Site) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event) {
	site.uiChan = uiChan
	site.lpUpdateChan = make(chan *LoadPoint)

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
func (site *Site) Run(interval time.Duration) {
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
		}
	}
}
