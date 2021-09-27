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
  host: 192.168.1.23
  cache: 2s
  usage: grid
- name: PvMeter
  type: rct
  host: 192.168.1.23
  cache: 2s
  usage: pv
- name: BatteryMeter
  type: rct
  host: 192.168.1.23
  cache: 2s
  usage: battery

** Limitations **
It is not allowed to provide different URIs or passwords for different lgess meters since always the
same hardware instance is accessed with the different usages.
*/

// RCTMeter implements the api.Meter interface
type RCTMeter struct {
	conn    *rct.Connection // connection with the RCT device
	usage    string         // grid, pv, battery
}

func init() {
	registry.Add("rct", NewRCTFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateRCT -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewRCTFromConfig creates an RCTMeter Meter from generic config
func NewRCTFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Host, Usage, Cache string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	cacheDuration,err:=time.ParseDuration(cc.Cache)
	if err!=nil {
		return nil, err
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewRCT(cc.Host, cc.Usage, cacheDuration)
}

// NewRCT creates an RCTMeter Meter
func NewRCT(ip, usage string, cache time.Duration) (api.Meter, error) {
	conn, err := rct.NewConnection(ip, cache)
	if err != nil {
		return nil, err
	}

	m := &RCTMeter{
		usage: strings.ToLower(usage),
		conn:    conn,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	//if m.usage == "grid" {
	totalEnergy = m.totalEnergy
	//}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}

	return decorateRCT(m, totalEnergy, batterySoC), nil
}

// CurrentPower implements the api.Meter interface
func (m *RCTMeter) CurrentPower() (float64, error) {
	switch m.usage {
	case "grid":
		res, err:=m.conn.QueryFloat32(rct.TotalGridPowerW)
		return float64(res), err

	case "pv":
		a, err:=m.conn.QueryFloat32(rct.SolarGenAPowerW)
		if err!=nil {	return 0, err }
		b, err:=m.conn.QueryFloat32(rct.SolarGenBPowerW)
		return float64(a+b), err

	case "battery":
		res, err:=m.conn.QueryFloat32(rct.BatteryPowerW)
		return float64(res), err

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// totalEnergy implements the api.MeterEnergy interface
func (m *RCTMeter) totalEnergy() (float64, error) {
	switch m.usage {
	case "grid":
		res, err:=m.conn.QueryFloat32(rct.TotalEnergyGridWh)
		return float64(res/1000), err

	case "pv":
		a, err:=m.conn.QueryFloat32(rct.TotalEnergySolarGenAWh)
		if err!=nil { return 0, err}
		b, err:=m.conn.QueryFloat32(rct.TotalEnergySolarGenBWh)
		return float64((a+b)/1000), err

	case "battery":
		in, err:=m.conn.QueryFloat32(rct.TotalEnergyBattInWh)
		if err!=nil { return 0, err}
		out, err:=m.conn.QueryFloat32(rct.TotalEnergyBattOutWh)
		return float64((in-out)/1000), err

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// batterySoC implements the api.Battery interface
func (m *RCTMeter) batterySoC() (float64, error) {
	res, err:=m.conn.QueryFloat32(rct.BatterySoC)
	return float64(res*100), err
}
