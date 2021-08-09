package meter

import (
	"errors"
	"fmt"
	"strings"
	"net/url"
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/meter/lgessv1"
)

// lgess is the LG ESS HOME meter
type LgEss struct {
	usage string 			// grid, pv, battery
	lgaccess *lgessv1.LgEssAccess 	// singleton controlling the acces to the LgEss data via the auth_key
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

	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	_, err := url.Parse(cc.URI)
	if err != nil {
		return nil, fmt.Errorf("%s is invalid: %s", cc.URI, err)
	}
	return NewLgEss(cc.URI, cc.Usage, cc.Password)
}

// NewLgEss creates an LgEss Meter
func NewLgEss(uri, usage, password string) (api.Meter, error) {

	lgaccess, err := lgessv1.GetInstance(uri, password)
	if err != nil {
		return nil, err
	}

	m := &LgEss{
		usage:      strings.ToLower(usage),
		lgaccess:   lgaccess,
	}

	//fmt.Printf("Usage:%v\r\nUri:%v\r\npassword:%v\r\n",m.usage, m.lgaccess.uri, m.lgaccess.password)

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

	switch m.usage {
	case "grid":
		power,err := m.lgaccess.GetGridPower()
		if err != nil {
			return 0, err
		}
		return power, nil
	case "pv":
		power,err := m.lgaccess.GetPvPower()
		if err != nil {
			return 0, err
		}
		return power, nil
	case "battery":
		power,err := m.lgaccess.GetBatPower()
		if err != nil {
			return 0, err
		}
		return power, nil
	}
	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
// @return float64 Current energy in kWh (of this day)
func (m *LgEss) totalEnergy() (float64, error) {

	switch m.usage {
	case "grid":
		energy,err := m.lgaccess.GetGridEnergy()
		if err != nil {
			return 0, err
		}
		return energy, nil
	case "pv":
		energy,err := m.lgaccess.GetPvEnergy()
		if err != nil {
			return 0, err
		}
		return energy, nil
	}
	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// batterySoC implements the api.Battery interface
// @return float64 The battery state of charge (SoC)
func (m *LgEss) batterySoC() (float64, error) {
	batSoC,err := m.lgaccess.GetBatSoC()
	if err != nil {
		return 0, err
	}
	return batSoC, nil
}

/* example json result of uri: /v1/user/essinfo/home
{
    "statistics":
    {
        "pcs_pv_total_power": "0",
        "batconv_power": "1287",
        "bat_use": "1",
        "bat_status": "2",
        "bat_user_soc": "59.3",
        "load_power": "1289",
        "ac_output_power": "10",
        "load_today": "0.0",
        "grid_power": "2",
        "current_day_self_consumption": "70.7",
        "current_pv_generation_sum": "62253",
        "current_grid_feed_in_energy": "18250"
    },
    "direction":
    {
        "is_direct_consuming_": "0",
        "is_battery_charging_": "0",
        "is_battery_discharging_": "1",
        "is_grid_selling_": "0",
        "is_grid_buying_": "0",
        "is_charging_from_grid_": "0",
        "is_discharging_to_grid_": "0"
    },
    "operation":
    {
        "status": "start",
        "mode": "1",
        "pcs_standbymode": "false",
        "drm_mode0": "0",
        "remote_mode": "0",
        "drm_control": "255"
    },
    "wintermode":
    {
        "winter_status": "off",
        "backup_status": "off"
    },
    "backupmode": "",
    "pcs_fault":
    {
        "pcs_status": "pcs_ok",
        "pcs_op_status": "pcs_run"
    },
    "heatpump":
    {
        "heatpump_protocol": "0",
        "heatpump_activate": "off",
        "current_temp": "0",
        "heatpump_working": "off"
    },
    "evcharger":
    {
        "ev_activate": "off",
        "ev_power": "0"
    },
    "gridWaitingTime": "0"
}


{
    "statistics":
    {
        "pcs_pv_total_power": "638",
        "batconv_power": "469",
        "bat_use": "1",
        "bat_status": "0",
        "bat_user_soc": "55.7",
        "load_power": "703",
        "ac_output_power": "10",
        "load_today": "0.0",
        "grid_power": "404",
        "current_day_self_consumption": "94.6",
        "current_pv_generation_sum": "31386",
        "current_grid_feed_in_energy": "1697"
    },
    "direction":
    {
        "is_direct_consuming_": "1",
        "is_battery_charging_": "0",
        "is_battery_discharging_": "0",
        "is_grid_selling_": "1",
        "is_grid_buying_": "0",
        "is_charging_from_grid_": "0",
        "is_discharging_to_grid_": "0"
    },
    "operation":
    {
        "status": "start",
        "mode": "1",
        "pcs_standbymode": "false",
        "drm_mode0": "0",
        "remote_mode": "0",
        "drm_control": "255"
    },
    "wintermode":
    {
        "winter_status": "off",
        "backup_status": "off"
    },
    "backupmode": "",
    "pcs_fault":
    {
        "pcs_status": "pcs_ok",
        "pcs_op_status": "pcs_run"
    },
    "heatpump":
    {
        "heatpump_protocol": "0",
        "heatpump_activate": "off",
        "current_temp": "0",
        "heatpump_working": "off"
    },
    "evcharger":
    {
        "ev_activate": "off",
        "ev_power": "0"
    },
    "gridWaitingTime": "0"
}


*/



