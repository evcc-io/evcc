package meter

import (
	"errors"
	"fmt"
	"strings"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/meter/lgpcs"
)

/**
 * This meter supports the LGESS HOME 8 and LGESS HOME 10 systems from LG with / without battery.
 *
 ** Usages **
 * The following usages are supported:
 * - grid    ... for reading the power imported or exported to the grid
 * - pv      ... for reading the power produced by the Photovoltaik
 * - battery ... for reading the power imported or exported to the battery
 *  *
 ** Example configuration **
 *
 * meters:
 * - name: GridMeter
 *   type: lgess
 *   usage: grid
 *   uri: https://192.168.1.23
 *   password: "DE200....."
 * - name: PvMeter
 *   type: lgess
 *   usage: pv
 * - name: BatteryMeter
 *   type: lgess
 *   usage: battery
 *
 *
 ** Limitations **
 * It is not allowed to provide different URIs or passwords for different lgess meters since always the
 * same hardware instance is accessed with the different usages.
 *
 * */

/**
 * Instance of one meter - multiple meter instances with different usages are allowed
 */
type LgEss struct {
	usage string 			// grid, pv, battery
	lgcom *lgpcs.LgPcsCom 	// singleton controlling the access to the LgEss data via the auth_key.
}

func init() {
	registry.Add("lgess", NewLgEssFromConfig)
}

//call "go generate" in the command line to automatically generate the decorators
//defined with the following comment
//go:generate go run ../cmd/tools/decorate.go -f decorateLgEss -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewLgEssFromConfig creates an LgEss Meter from generic config
func NewLgEssFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Usage, Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}
	return NewLgEss(cc.URI, cc.Usage, cc.Password)
}

// NewLgEss creates an LgEss Meter
func NewLgEss(uri, usage, password string) (api.Meter, error) {

	lgpcs, err := lgpcs.GetInstance(uri, password)
	if err != nil {
		return nil, err
	}

	m := &LgEss{
		usage:   strings.ToLower(usage),
		lgcom:   lgpcs,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "grid" || m.usage == "pv" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}
	return decorateLgEss(m, totalEnergy, batterySoC), nil
}

// CurrentPower implements the api.Meter interface
// @return float64 current power in W
func (m *LgEss) CurrentPower() (float64, error) {

	data, err := m.lgcom.GetData()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return data.GridPower, nil
	case "pv":
		return data.PvPower, nil
	case "battery":
		return data.BatPower, nil
	}
	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
// @return float64 Current energy in kWh (of this day)
func (m *LgEss) totalEnergy() (float64, error) {

	data, err := m.lgcom.GetData()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return data.GridEnergy, nil
	case "pv":
		return data.PvEnergy, nil
	}
	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// batterySoC implements the api.Battery interface
// @return float64 The battery state of charge (SoC)
func (m *LgEss) batterySoC() (float64, error) {
	data, err := m.lgcom.GetData()
	if err != nil {
		return 0, err
	}
	return data.BatSoC, nil
}
