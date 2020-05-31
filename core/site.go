package core

import (
	"math"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	"github.com/avast/retry-go"
	"github.com/pkg/errors"
)

type Site struct {
	sync.Mutex                    // guard status
	triggerChan chan struct{}     // API updates
	uiChan      chan<- util.Param // client push messages

	Title         string         // UI title
	Voltage       float64        // Operating voltage. 230V for Germany.
	ResidualPower float64        // PV meter only: household usage. Grid meter: household safety margin
	Mode          api.ChargeMode // Charge mode, guarded by mutex
	Meters        MetersConfig   // Meter references

	// meters
	gridMeter    api.Meter // Grid usage meter
	pvMeter      api.Meter // PV generation meter
	batteryMeter api.Meter // Battery charging meter

	loadPoints []*LoadPoint // Loadpoints

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
	loadPoints []*LoadPoint,
) *Site {
	site := NewSite()
	util.DecodeOther(log, other, &site)

	// workaround mapstructure
	if site.Mode == "0" {
		site.Mode = api.ModeOff
	}

	if site.Meters.PVMeterRef == "" && site.Meters.GridMeterRef == "" {
		log.FATAL.Fatal("config: missing either pv or grid meter")
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

	if site.pvMeter == nil && site.gridMeter == nil {
		log.FATAL.Fatal("missing either pv or grid meter")
	}

	site.loadPoints = loadPoints
	for _, lp := range loadPoints {
		lp.Site = site
	}

	return site
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	lp := &Site{
		triggerChan: make(chan struct{}, 1),
		Mode:        api.ModeOff,
		Voltage:     230, // V
	}

	return lp
}

// SiteConfiguration is the loadpoint feature structure
type SiteConfiguration struct {
	Mode         string                   `json:"mode"`
	GridMeter    bool                     `json:"gridMeter"`
	PVMeter      bool                     `json:"pvMeter"`
	BatteryMeter bool                     `json:"batteryMeter"`
	LoadPoints   []LoadpointConfiguration `json:"loadPoints"`
}

// LoadpointConfiguration is the loadpoint feature structure
type LoadpointConfiguration struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Phases      int64  `json:"phases"`
	MinCurrent  int64  `json:"minCurrent"`
	MaxCurrent  int64  `json:"maxCurrent"`
	ChargeMeter bool   `json:"chargeMeter"`
	SoC         bool   `json:"soc"`
	SoCCapacity int64  `json:"socCapacity"`
	SoCTitle    string `json:"socTitle"`
}

// Configuration returns meter configuration
func (lp *Site) Configuration() SiteConfiguration {
	c := SiteConfiguration{
		Mode:         string(lp.GetMode()),
		GridMeter:    lp.gridMeter != nil,
		PVMeter:      lp.pvMeter != nil,
		BatteryMeter: lp.batteryMeter != nil,
	}

	for _, lp := range lp.loadPoints {
		l := LoadpointConfiguration{
			Name:        lp.Name,
			Title:       lp.Title,
			Phases:      lp.Phases,
			MinCurrent:  lp.MinCurrent,
			MaxCurrent:  lp.MaxCurrent,
			ChargeMeter: lp.hasChargeMeter(),
		}

		if lp.vehicle != nil {
			l.SoC = true
			l.SoCCapacity = lp.vehicle.Capacity()
			l.SoCTitle = lp.vehicle.Title()
		}

		c.LoadPoints = append(c.LoadPoints, l)
	}

	return c
}

// DumpConfig site configuration
func (lp *Site) DumpConfig() {
	log.INFO.Println("site config:")
	log.INFO.Printf("  mode: %s", lp.GetMode())
	log.INFO.Printf("  grid %s", presence[lp.gridMeter != nil])
	log.INFO.Printf("  pv %s", presence[lp.pvMeter != nil])
	log.INFO.Printf("  battery %s", presence[lp.batteryMeter != nil])

	for i, lp := range lp.loadPoints {
		log.INFO.Printf("loadpoint %d config:", i)

		log.INFO.Printf("  name %s", lp.Name)
		log.INFO.Printf("  vehicle %s", presence[lp.vehicle != nil])
		log.INFO.Printf("  charge %s", presence[lp.hasChargeMeter()])

		_, power := lp.charger.(api.Meter)
		_, energy := lp.charger.(api.ChargeRater)
		_, timer := lp.charger.(api.ChargeTimer)

		log.INFO.Println("  charger config:")
		log.INFO.Printf("    power %s", presence[power])
		log.INFO.Printf("    energy %s", presence[energy])
		log.INFO.Printf("    timer %s", presence[timer])
	}
}

// Update triggers loadpoint to run main control loop and push messages to UI socket
func (lp *Site) Update() {
	select {
	case lp.triggerChan <- struct{}{}: // non-blocking send
	default:
		log.WARN.Printf("update blocked")
	}
}

// GetMode returns loadpoint charge mode
func (lp *Site) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()
	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *Site) SetMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	log.INFO.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.Mode != mode {
		for _, lp := range lp.loadPoints {
			lp.resetGuard()
		}

		lp.Mode = mode
	}

	lp.Update()
}

// publish sends values to UI and databases
func (lp *Site) publish(key string, val interface{}) {
	lp.uiChan <- util.Param{
		Key: key,
		Val: val,
	}
}

// updateMeter updates and publishes single meter
func (lp *Site) updateMeter(name string, meter api.Meter, power *float64) error {
	value, err := meter.CurrentPower()
	if err != nil {
		return err
	}

	*power = value // update value if no error

	log.DEBUG.Printf("%s power: %.1fW", name, *power)
	lp.publish(name+"Power", *power)

	return nil
}

// updateMeter updates and publishes single meter
func (lp *Site) updateMeters() (err error) {
	retryMeter := func(s string, m api.Meter, f *float64) {
		if m != nil {
			e := retry.Do(func() error {
				return lp.updateMeter(s, m, f)
			}, retry.Attempts(3))

			if e != nil {
				err = errors.Wrapf(e, "updating %s meter", s)
				log.ERROR.Println(err)
			}
		}
	}

	// read PV meter before charge meter
	retryMeter("grid", lp.gridMeter, &lp.gridPower)
	retryMeter("pv", lp.pvMeter, &lp.pvPower)
	retryMeter("battery", lp.batteryMeter, &lp.batteryPower)

	return err
}

func consumedPower(pv, battery, grid float64) float64 {
	return math.Abs(pv) + battery + grid
}

// consumedPower estimates how much power the charger might have consumed given it was the only load
func (lp *Site) consumedPower() float64 {
	return consumedPower(lp.pvPower, lp.batteryPower, lp.gridPower)
}

// sitePower returns the net power exported by the site minus a residual margin.
// negative values mean grid: export, battery: charging
func (lp *Site) sitePower() (float64, error) {
	if err := lp.updateMeters(); err != nil {
		return 0, err
	}

	sitePower := lp.gridPower + lp.batteryPower + lp.ResidualPower
	return sitePower, nil
}

func (lp *Site) update() error {
	mode := lp.GetMode()
	lp.publish("mode", string(mode))

	availablePower, err := lp.sitePower()
	if err != nil {
		return err
	}

	log.DEBUG.Printf("site power: %.0fW", availablePower)

	for _, lp := range lp.loadPoints {
		usedPower := lp.update(mode, availablePower)
		remainingPower := availablePower + usedPower
		log.DEBUG.Printf("%s remaining power: %.0fW = %.0fW - %.0fW", lp.Name, remainingPower, availablePower, usedPower)
		availablePower = remainingPower
	}

	return nil
}

// Run is the loadpoint main control loop. It reacts to trigger events by
// updating measurements and executing control logic.
func (lp *Site) Run(uiChan chan<- util.Param, pushChan chan<- push.Event, interval time.Duration) {
	lp.uiChan = uiChan
	for _, _lp := range lp.loadPoints {
		_lp.Prepare(uiChan, pushChan)
		_lp.Voltage = lp.Voltage
	}

	ticker := time.NewTicker(interval)
	lp.triggerChan <- struct{}{} // start immediately

	for {
		select {
		case <-ticker.C:
			if lp.update() != nil {
				lp.triggerChan <- struct{}{} // restart immediately
			}
		case <-lp.triggerChan:
			lp.update()
			ticker.Stop()
			ticker = time.NewTicker(interval)
		}
	}
}
