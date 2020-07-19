package core

import (
	"sync"
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
	Update(api.ChargeMode, float64)
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	sync.Mutex                    // guard status
	triggerChan chan struct{}     // API updates
	uiChan      chan<- util.Param // client push messages
	log         *util.Logger

	// configuration
	Title         string         `mapstructure:"title"`         // UI title
	Voltage       float64        `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower float64        `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Mode          api.ChargeMode `mapstructure:"mode"`          // Charge mode, guarded by mutex
	Meters        MetersConfig   // Meter references

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
	util.DecodeOther(log, other, &site)

	// workaround mapstructure
	if site.Mode == "0" {
		site.Mode = api.ModeOff
	}

	// configure meter from references
	// if site.Meters.PVMeterRef == "" && site.Meters.GridMeterRef == "" {
	// 	site.log.FATAL.Fatal("config: missing either pv or grid meter")
	// }
	if site.Meters.GridMeterRef == "" {
		site.log.FATAL.Fatal("config: missing grid meter")
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

	// convert loadpoints to interfaces
	for _, lp := range loadpoints {
		site.loadpoints = append(site.loadpoints, lp)
	}

	Voltage = site.Voltage

	return site
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	lp := &Site{
		log:         util.NewLogger("core"),
		triggerChan: make(chan struct{}, 1),
		Mode:        api.ModeOff,
		Voltage:     230, // V
	}

	return lp
}

// SiteConfiguration contains the global site configuration
type SiteConfiguration struct {
	Title        string                   `json:"title"`
	Mode         string                   `json:"mode"`
	GridMeter    bool                     `json:"gridMeter"`
	PVMeter      bool                     `json:"pvMeter"`
	BatteryMeter bool                     `json:"batteryMeter"`
	LoadPoints   []LoadpointConfiguration `json:"loadpoints"`
}

// LoadpointConfiguration is the loadpoint feature structure
type LoadpointConfiguration struct {
	Title       string `json:"title"`
	Phases      int64  `json:"phases"`
	MinCurrent  int64  `json:"minCurrent"`
	MaxCurrent  int64  `json:"maxCurrent"`
	ChargeMeter bool   `json:"chargeMeter"`
	SoC         bool   `json:"soc"`
	SoCCapacity int64  `json:"socCapacity"`
	SoCTitle    string `json:"socTitle"`
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
		Mode:         string(site.GetMode()),
		GridMeter:    site.gridMeter != nil,
		PVMeter:      site.pvMeter != nil,
		BatteryMeter: site.batteryMeter != nil,
	}

	for _, lp := range site.loadpoints {
		lpc := LoadpointConfiguration{
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
		}

		c.LoadPoints = append(c.LoadPoints, lpc)
	}

	return c
}

// DumpConfig site configuration
func (site *Site) DumpConfig() {
	site.log.INFO.Println("site config:")
	site.log.INFO.Printf("  mode: %s", site.GetMode())
	site.log.INFO.Printf("  grid %s", presence[site.gridMeter != nil])
	site.log.INFO.Printf("  pv %s", presence[site.pvMeter != nil])
	site.log.INFO.Printf("  battery %s", presence[site.batteryMeter != nil])

	for i, lp := range site.loadpoints {
		lp.log.INFO.Printf("loadpoint %d config:", i)

		lp.log.INFO.Printf("  vehicle %s", presence[lp.vehicle != nil])
		lp.log.INFO.Printf("  charge %s", presence[lp.hasChargeMeter()])

		charger := lp.handler.(*ChargerHandler).charger
		_, power := charger.(api.Meter)
		_, energy := charger.(api.ChargeRater)
		_, timer := charger.(api.ChargeTimer)

		lp.log.INFO.Println("  charger config:")
		lp.log.INFO.Printf("    power %s", presence[power])
		lp.log.INFO.Printf("    energy %s", presence[energy])
		lp.log.INFO.Printf("    timer %s", presence[timer])
	}
}

// Update triggers loadpoint to run main control loop and push messages to UI socket
func (site *Site) Update() {
	select {
	case site.triggerChan <- struct{}{}: // non-blocking send
	default:
		site.log.WARN.Printf("update blocked")
	}
}

// GetMode returns loadpoint charge mode
func (site *Site) GetMode() api.ChargeMode {
	site.Lock()
	defer site.Unlock()
	return site.Mode
}

// SetMode sets loadpoint charge mode
func (site *Site) SetMode(mode api.ChargeMode) {
	site.Lock()
	defer site.Unlock()

	site.log.INFO.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if site.Mode != mode {
		site.Mode = mode
		site.Update()
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

	site.log.DEBUG.Printf("%s power: %.1fW", name, *power)
	site.publish(name+"Power", *power)

	return nil
}

// updateMeter updates and publishes single meter
func (site *Site) updateMeters() (err error) {
	retryMeter := func(s string, m api.Meter, f *float64) {
		if m != nil {
			e := retry.Do(func() error {
				return site.updateMeter(s, m, f)
			}, retryOptions...)

			if e != nil {
				err = errors.Wrapf(e, "updating %s meter", s)
				site.log.ERROR.Println(err)
			}
		}
	}

	// read PV meter before charge meter
	retryMeter("grid", site.gridMeter, &site.gridPower)
	retryMeter("pv", site.pvMeter, &site.pvPower)
	retryMeter("battery", site.batteryMeter, &site.batteryPower)

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
	mode := site.GetMode()
	site.publish("mode", string(mode))

	if sitePower, err := site.sitePower(); err == nil {
		lp.Update(mode, sitePower)
	}
}

// Prepare attaches communication channels to site and loadpoints
func (site *Site) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event) {
	site.uiChan = uiChan

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

		lp.Prepare(lpUIChan, lpPushChan)
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
	// update ticker
	ticker := time.NewTicker(interval)
	site.triggerChan <- struct{}{} // start immediately

	loadpointChan := make(chan Updater)
	go site.loopLoadpoints(loadpointChan)

	for {
		select {
		case <-ticker.C:
			site.update(<-loadpointChan)
		case <-site.triggerChan:
			for range site.loadpoints {
				site.update(<-loadpointChan)
			}
			ticker.Stop()
			ticker = time.NewTicker(interval)
		}
	}
}
