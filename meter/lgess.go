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

/*
This meter supports the LGESS HOME 8, LGESS HOME 10 and LGESS HOME 15 systems from LG with / without battery.


** Usages **
The following usages are supported:
- grid    ... for reading the power imported or exported to the grid
- pv      ... for reading the power produced by the pv
- battery ... for reading the power imported or exported to the battery

** Example configuration **
meters:
- name: GridMeter
  type: template
  template: lg-ess-home-15
  usage: grid
  uri: https://192.168.1.23
  password: "DE200....."
- name: PvMeter
  type: template
  template: lg-ess-home-15
  usage: pv
- name: BatteryMeter
  type: template
  template: lg-ess-home-15
  usage: battery

** Limitations **
It is not allowed to provide different URIs or passwords for different lgess meters since always the
same hardware instance is accessed with the different usages.
*/

// LgEss implements the api.Meter interface
type LgEss struct {
	usage string     // grid, pv, battery
	conn  *lgpcs.Com // communication with the lgpcs device
}

func init() {
	registry.Add("lgess8", NewLgEss8FromConfig)
	registry.Add("lgess15", NewLgEss15FromConfig)
}

//go:generate decorate -f decorateLgEss -b *LgEss -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

func NewLgEss8FromConfig(other map[string]interface{}) (api.Meter, error) {
	return NewLgEssFromConfig(other, lgpcs.LgEss8)
}

func NewLgEss15FromConfig(other map[string]interface{}) (api.Meter, error) {
	return NewLgEssFromConfig(other, lgpcs.LgEss15)
}

// NewLgEssFromConfig creates an LgEss Meter from generic config
func NewLgEssFromConfig(other map[string]interface{}, essType lgpcs.Model) (api.Meter, error) {
	cc := struct {
		capacity               `mapstructure:",squash"`
		URI, Usage             string
		Registration, Password string
		Cache                  time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewLgEss(cc.URI, cc.Usage, cc.Registration, cc.Password, cc.Cache, cc.capacity.Decorator(), essType)
}

// NewLgEss creates an LgEss Meter
func NewLgEss(uri, usage, registration, password string, cache time.Duration, capacity func() float64, essType lgpcs.Model) (api.Meter, error) {
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

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = m.batterySoc
	}

	return decorateLgEss(m, totalEnergy, batterySoc, capacity), nil
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
