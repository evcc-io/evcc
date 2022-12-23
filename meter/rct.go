package meter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"

	"github.com/mlnoga/rct"
)

/*
This meter supports devices implementing the RCT communication protocol, e.g. the RCT PS 6.0 with / without battery.

** Usages **
The following usages are supported:
- grid    ... for reading the power imported or exported to the grid
- pv      ... for reading the power produced by the pv
- battery ... for reading the power imported or exported to the battery

** Example configuration **
meters:
- name: GridMeter
  type: rct
  uri: 192.168.1.23
  cache: 2s
  usage: grid
- name: PvMeter
  type: rct
  uri: 192.168.1.23
  cache: 2s
  usage: pv
- name: BatteryMeter
  type: rct
  uri: 192.168.1.23
  cache: 2s
  usage: battery
*/

// RCT implements the api.Meter interface
type RCT struct {
	conn  *rct.Connection // connection with the RCT device
	usage string          // grid, pv, battery
}

func init() {
	registry.Add("rct", NewRCTFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateRCT -b *RCT -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)"

// NewRCTFromConfig creates an RCT from generic config
func NewRCTFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Uri, Usage string
		Cache      time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewRCT(cc.Uri, cc.Usage, cc.Cache)
}

// NewRCT creates an RCT meter
func NewRCT(uri, usage string, cache time.Duration) (api.Meter, error) {
	conn, err := rct.NewConnection(uri, cache)
	if err != nil {
		return nil, err
	}

	m := &RCT{
		usage: strings.ToLower(usage),
		conn:  conn,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if usage == "grid" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = m.batterySoc
	}

	return decorateRCT(m, totalEnergy, batterySoc), nil
}

// CurrentPower implements the api.Meter interface
func (m *RCT) CurrentPower() (float64, error) {
	switch m.usage {
	case "grid":
		res, err := m.conn.QueryFloat32(rct.TotalGridPowerW)
		return float64(res), err

	case "pv":
		a, err := m.conn.QueryFloat32(rct.SolarGenAPowerW)
		if err != nil {
			return 0, err
		}
		b, err := m.conn.QueryFloat32(rct.SolarGenBPowerW)
		return float64(a + b), err

	case "battery":
		res, err := m.conn.QueryFloat32(rct.BatteryPowerW)
		return float64(res), err

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// totalEnergy implements the api.MeterEnergy interface
func (m *RCT) totalEnergy() (float64, error) {
	switch m.usage {
	case "grid":
		res, err := m.conn.QueryFloat32(rct.TotalEnergyGridWh)
		return float64(res / 1000), err

	case "pv":
		a, err := m.conn.QueryFloat32(rct.TotalEnergySolarGenAWh)
		if err != nil {
			return 0, err
		}
		b, err := m.conn.QueryFloat32(rct.TotalEnergySolarGenBWh)
		return float64((a + b) / 1000), err

	case "battery":
		in, err := m.conn.QueryFloat32(rct.TotalEnergyBattInWh)
		if err != nil {
			return 0, err
		}
		out, err := m.conn.QueryFloat32(rct.TotalEnergyBattOutWh)
		return float64((in - out) / 1000), err

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// batterySoc implements the api.Battery interface
func (m *RCT) batterySoc() (float64, error) {
	res, err := m.conn.QueryFloat32(rct.BatterySoC)
	return float64(res * 100), err
}
