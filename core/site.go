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

//go:generate mockgen -package mock -destination ../mock/mock_loadpoint.go github.com/andig/evcc/core LoadPointer

// LoadPointer abstracts the LoadPoint implementation for testing
type LoadPointer interface {
	Update(api.ChargeMode, float64) float64
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Site struct {
	sync.Mutex                    // guard status
	triggerChan chan struct{}     // API updates
	uiChan      chan<- util.Param // client push messages
	log         *util.Logger

	// configuration
	Title         string         `mapstructure:"title"`         // UI title
	VoltageRef    float64        `mapstructure:"voltage"`       // Operating voltage. 230V for Germany.
	ResidualPower float64        `mapstructure:"residualPower"` // PV meter only: household usage. Grid meter: household safety margin
	Mode          api.ChargeMode `mapstructure:"mode"`          // Charge mode, guarded by mutex
	Meters        MetersConfig   // Meter references

	// meters
	gridMeter    api.Meter // Grid usage meter
	pvMeter      api.Meter // PV generation meter
	batteryMeter api.Meter // Battery charging meter

	loadPoints []LoadPointer // Loadpoints

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
	for _, lp := range loadPoints {
		site.loadPoints = append(site.loadPoints, lp)
	}

	// validate single voltage
	if Voltage != 0 && Voltage != site.VoltageRef {
		site.log.FATAL.Fatal("config: only single voltage allowed")
	}

	Voltage = site.VoltageRef

	return site
}

// NewSite creates a Site with sane defaults
func NewSite() *Site {
	lp := &Site{
		log:         util.NewLogger("core"),
		triggerChan: make(chan struct{}, 1),
		Mode:        api.ModeOff,
		VoltageRef:  230, // V
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
func (lp *Site) LoadPoints() []*LoadPoint {
	res := make([]*LoadPoint, 0, len(lp.loadPoints))
	for _, lp := range lp.loadPoints {
		res = append(res, lp.(*LoadPoint))
	}
	return res
}

// Configuration returns meter configuration
func (lp *Site) Configuration() SiteConfiguration {
	c := SiteConfiguration{
		Title:        lp.Title,
		Mode:         string(lp.GetMode()),
		GridMeter:    lp.gridMeter != nil,
		PVMeter:      lp.pvMeter != nil,
		BatteryMeter: lp.batteryMeter != nil,
	}

	for _, lptr := range lp.loadPoints {
		lp := lptr.(*LoadPoint)

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
func (lp *Site) DumpConfig() {
	lp.log.INFO.Println("site config:")
	lp.log.INFO.Printf("  mode: %s", lp.GetMode())
	lp.log.INFO.Printf("  grid %s", presence[lp.gridMeter != nil])
	lp.log.INFO.Printf("  pv %s", presence[lp.pvMeter != nil])
	lp.log.INFO.Printf("  battery %s", presence[lp.batteryMeter != nil])

	for i, lptr := range lp.loadPoints {
		lp := lptr.(*LoadPoint)
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
func (lp *Site) Update() {
	select {
	case lp.triggerChan <- struct{}{}: // non-blocking send
	default:
		lp.log.WARN.Printf("update blocked")
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

	lp.log.INFO.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.Mode != mode {
		lp.Mode = mode
		lp.Update()
	}
}

// publish sends values to UI and databases
func (lp *Site) publish(key string, val interface{}) {
	// test helper
	if lp.uiChan == nil {
		return
	}

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

	lp.log.DEBUG.Printf("%s power: %.1fW", name, *power)
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
				lp.log.ERROR.Println(err)
			}
		}
	}

	// read PV meter before charge meter
	retryMeter("grid", lp.gridMeter, &lp.gridPower)
	retryMeter("pv", lp.pvMeter, &lp.pvPower)
	retryMeter("battery", lp.batteryMeter, &lp.batteryPower)

	return err
}

// consumedPower estimates how much power the charger might have consumed given it was the only load
// func (lp *Site) consumedPower() float64 {
// 	return consumedPower(lp.pvPower, lp.batteryPower, lp.gridPower)
// }

// sitePower returns the net power exported by the site minus a residual margin.
// negative values mean grid: export, battery: charging
func (lp *Site) sitePower() (float64, error) {
	if err := lp.updateMeters(); err != nil {
		return 0, err
	}

	sitePower := sitePower(lp.gridPower, lp.batteryPower, lp.ResidualPower)
	lp.log.DEBUG.Printf("site power: %.0fW", sitePower)

	return sitePower, nil
}

func (lp *Site) update() error {
	mode := lp.GetMode()
	lp.publish("mode", string(mode))

	sitePower, err := lp.sitePower()
	if err != nil {
		return err
	}

	for idx, loadPoint := range lp.loadPoints {
		usedPower := loadPoint.Update(mode, sitePower)
		remainingPower := sitePower + usedPower
		lp.log.DEBUG.Printf("lp-%d remaining power: %.0fW = %.0fW - %.0fW", idx+1, remainingPower, sitePower, usedPower)
		sitePower = remainingPower
	}

	return nil
}

// Prepare attaches communication channels to site and loadpoints
func (lp *Site) Prepare(uiChan chan<- util.Param, pushChan chan<- push.Event) {
	lp.uiChan = uiChan

	for id, loadPoint := range lp.loadPoints {
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

		loadPoint.(*LoadPoint).Prepare(lpUIChan, lpPushChan)
	}
}

// Run is the main control loop. It reacts to trigger events by
// updating measurements and executing control logic.
func (lp *Site) Run(interval time.Duration) {
	// update ticker
	ticker := time.NewTicker(interval)
	lp.triggerChan <- struct{}{} // start immediately

	for {
		select {
		case <-ticker.C:
			if lp.update() != nil {
				lp.triggerChan <- struct{}{} // restart immediately
			}
		case <-lp.triggerChan:
			_ = lp.update()
			ticker.Stop()
			ticker = time.NewTicker(interval)
		}
	}
}
