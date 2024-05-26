package meter

import (
	"math"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/foxesscloud"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// FoxESSCloudH3 meter implementation
type FoxESSCloudH3 struct {
	battery   battery
	foxess    *foxesscloud.FoxESSCloudAPI
	cached    Cached
	usage, sn string
}

type Cached struct {
	GetRealTimeData func() (*foxesscloud.GetDeviceRealTimeData, error)
}

func init() {
	registry.Add("fox-ess-cloud-h3", NewFoxESSCloudH3FromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateFoxESSCloudH3 -b *FoxESSCloudH3 -r api.Meter -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)"  -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

// NewFoxESSCloudH3FromConfig creates a FoxESSCloudH3 meter from config
func NewFoxESSCloudH3FromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Usage, Key, Sn string
		Battery        battery `mapstructure:",squash"`
	}{
		Battery: battery{
			MinSoc: 10,
			MaxSoc: 95,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFoxESSCloudH3(cc.Usage, cc.Key, cc.Sn, cc.Battery)
}

// NewFoxESSCloudH3 creates FoxESSCloudH3 meter
func NewFoxESSCloudH3(usage, key, sn string, battery battery) (api.Meter, error) {
	logger := util.NewLogger("foxess-cloud-h3").Redact(key)

	helper := &request.Helper{
		Client: &http.Client{
			Timeout:   time.Minute,
			Transport: request.NewTripper(logger, transport.Default()),
		},
	}

	foxess := foxesscloud.NewFoxESSCloudAPI(key, helper, logger)

	m := &FoxESSCloudH3{
		battery: battery,
		foxess:  foxess,
		usage:   usage,
		sn:      sn,

		cached: Cached{
			// https://www.foxesscloud.com/public/i18n/en/OpenApiDocument.html#4
			GetRealTimeData: provider.Cached(func() (*foxesscloud.GetDeviceRealTimeData, error) {
				return foxess.GetDeviceRealTimeData(sn, []string{
					"meterPower",
					"pvPower",
					"batDischargePower",
					"batChargePower",
					"ResidualEnergy",
					"SoC",
					"meterPowerR",
					"meterPowerS",
					"meterPowerT",
				})
			}, 1*time.Minute),
		},
	}

	// decorate api.PhasePowers
	var Powers func() (float64, float64, float64, error)
	if m.usage == "grid" {
		Powers = m.dPowers
	}

	// decorate api.MeterPower
	var TotalEnergy func() (float64, error)
	if m.usage == "battery" {
		TotalEnergy = m.dTotalEnergy
	}

	// decorate api.Battery
	var Soc func() (float64, error)
	if m.usage == "battery" {
		Soc = m.dSoc
	}

	// decorate api.BatteryController
	var SetBatteryMode func(api.BatteryMode) error
	if m.usage == "battery" {
		SetBatteryMode = m.dSetBatteryMode
	}
	return decorateFoxESSCloudH3(m, Powers, TotalEnergy, Soc, SetBatteryMode), nil
}

var _ api.Meter = (*FoxESSCloudH3)(nil)

// CurrentPower implements the api.Meter interface
func (m *FoxESSCloudH3) CurrentPower() (float64, error) {
	real, err := m.cached.GetRealTimeData()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return (*real.Data.MeterPower) * 1e3, nil // W
	case "pv":
		return *real.Data.PvPower * 1e3, nil // W
	case "battery":
		return (*real.Data.BatDischargePower - *real.Data.BatChargePower) * 1e3, nil // W
	default:
		return 0, api.ErrNotAvailable
	}
}

// Currents implements the api.PhasePowers interface
func (m *FoxESSCloudH3) dPowers() (float64, float64, float64, error) {
	real, err := m.cached.GetRealTimeData()
	if err != nil {
		return 0, 0, 0, err
	}

	switch m.usage {
	case "grid":
		return *real.Data.MeterPowerR * 1e3, *real.Data.MeterPowerS * 1e3, *real.Data.MeterPowerT * 1e3, nil // W
	default:
		return 0, 0, 0, api.ErrNotAvailable
	}
}

// dTotalEnergy implements the api.MeterEnergy interface
func (m *FoxESSCloudH3) dTotalEnergy() (float64, error) {
	real, err := m.cached.GetRealTimeData()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "battery":
		return (*real.Data.ResidualEnergy) / 1e2, nil // kWh
	default:
		return 0, api.ErrNotAvailable
	}
}

// dSoc implements the api.Battery interface
func (m *FoxESSCloudH3) dSoc() (float64, error) {
	real, err := m.cached.GetRealTimeData()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "battery":
		return float64(*real.Data.SoC), nil // %

	default:
		return 0, api.ErrNotAvailable
	}
}

// dSetBatteryMode implements the api.BatteryController interface
func (m *FoxESSCloudH3) dSetBatteryMode(mode api.BatteryMode) error {
	return m.battery.LimitController(m.getSoC, func(limit float64) error {
		return m.foxess.SetDeviceMinSoc(m.sn, uint8(limit), uint8(limit))
	})(mode)
}

// getSoC retrieves current SoC
func (m *FoxESSCloudH3) getSoC() (float64, error) {
	real, err := m.cached.GetRealTimeData()
	if err != nil {
		return 0, err
	}
	soc := math.Round(float64(*real.Data.SoC) + 0.5) // .5 ensures we round up
	return soc, nil
}