package meter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/lgpcs"
	"github.com/evcc-io/evcc/util"
)

// LgEss implements the api.Meter interface
type LgEss struct {
	usage string     // grid, pv, battery
	conn  *lgpcs.Com // communication with the lgpcs device
}

func init() {
	registry.Add("lgess8", NewLgEss8FromConfig)
	registry.Add("lgess15", NewLgEss15FromConfig)
}

//go:generate go tool decorate -f decorateLgEss -b *LgEss -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.BatteryCapacity,Capacity,func() float64"

func NewLgEss8FromConfig(other map[string]interface{}) (api.Meter, error) {
	return NewLgEssFromConfig(other, lgpcs.LgEss8)
}

func NewLgEss15FromConfig(other map[string]interface{}) (api.Meter, error) {
	return NewLgEssFromConfig(other, lgpcs.LgEss15)
}

// NewLgEssFromConfig creates an LgEss Meter from generic config
func NewLgEssFromConfig(other map[string]interface{}, essType lgpcs.Model) (api.Meter, error) {
	cc := struct {
		batteryCapacity        `mapstructure:",squash"`
		batterySocLimits       `mapstructure:",squash"`
		URI, Usage             string
		Registration, Password string
		Cache                  time.Duration
	}{
		batterySocLimits: batterySocLimits{
			MinSoc: 20,
			MaxSoc: 95,
		},
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewLgEss(cc.URI, cc.Usage, cc.Registration, cc.Password, cc.Cache, cc.batteryCapacity.Decorator(), cc.batterySocLimits, essType)
}

// NewLgEss creates an LgEss Meter
func NewLgEss(uri, usage, registration, password string, cache time.Duration, capacity func() float64, batterySocLimits batterySocLimits, essType lgpcs.Model) (api.Meter, error) {
	conn, err := lgpcs.GetInstance(uri, registration, password, cache, essType)
	if err != nil {
		return nil, err
	}

	m := &LgEss{
		usage: strings.ToLower(usage),
		conn:  conn,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "grid" && essType != lgpcs.LgEss15 {
		totalEnergy = m.totalEnergy
	}

	// decorate battery
	var batterySoc func() (float64, error)
	var setBatteryMode func(api.BatteryMode) error
	if usage == "battery" {
		batterySoc = m.batterySoc
		if version, err := conn.GetFirmwareVersion(); err == nil && version >= 7433 {
			setBatteryMode = m.batteryMode(batterySocLimits)
		}
	}

	return decorateLgEss(m, totalEnergy, batterySoc, setBatteryMode, capacity), nil
}

// CurrentPower implements the api.Meter interface
func (m *LgEss) CurrentPower() (float64, error) {
	data, err := m.conn.Data()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return data.GetGridPower(), nil
	case "pv":
		return data.GetPvTotalPower(), nil
	case "battery":
		return data.GetBatConvPower(), nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// totalEnergy implements the api.MeterEnergy interface
func (m *LgEss) totalEnergy() (float64, error) {
	data, err := m.conn.Data()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return data.GetCurrentGridFeedInEnergy() / 1e3, nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// batterySoc implements the api.Battery interface
func (m *LgEss) batterySoc() (float64, error) {
	data, err := m.conn.Data()
	if err != nil {
		return 0, err
	}

	return data.GetBatUserSoc(), nil
}

// batteryMode implements the api.BatteryController interface
func (m *LgEss) batteryMode(batterySocLimits batterySocLimits) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			// firmeware bug: battery not discharging after hold mode
			// if battery is sleeping, wake up with charging for 10sec
			m.conn.BatteryMode("on", 100, true)
			time.Sleep(10 * time.Second)
			// now turn Battery discharge on
			return m.conn.BatteryMode("on", int(batterySocLimits.MinSoc), true)
		case api.BatteryHold:
			soc, err := m.batterySoc()
			if err != nil {
				return err
			}
			// soc needs to be the next higher int value to stop discharging immediately
			// example: batterySoc=50.7 -> set 51
			if int(soc)+1 < 100 {
				soc++
			}
			return m.conn.BatteryMode("on", int(soc), true)
		case api.BatteryCharge:
			return m.conn.BatteryMode("on", int(batterySocLimits.MaxSoc), true)
		default:
			return api.ErrNotAvailable
		}
	}
}
