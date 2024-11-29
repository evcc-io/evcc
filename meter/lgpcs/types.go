package lgpcs

import "math"

// Models
type Model int

const (
	LgEss8  = 0 // lgess 8/10
	LgEss15 = 1 // lgess 15
)

// data in the format expected by the accessing (lgess) module
type EssData interface {
	GetGridPower() float64               // in [W]
	GetPvTotalPower() float64            // in [W]
	GetBatConvPower() float64            // in [W]
	GetBatUserSoc() float64              // in [%]
	GetCurrentGridFeedInEnergy() float64 // in [Wh]
	GetCurrentPvGenerationSum() float64  // in [Wh]
}

type EssData8 struct {
	GridPower               float64 `json:"grid_power,string"`
	PvTotalPower            float64 `json:"pcs_pv_total_power,string"`
	BatConvPower            float64 `json:"batconv_power,string"`
	BatUserSoc              float64 `json:"bat_user_soc,string"`
	CurrentGridFeedInEnergy float64 `json:"current_grid_feed_in_energy,string"`
	CurrentPvGenerationSum  float64 `json:"current_pv_generation_sum,string"`
}

type MeterResponse8 struct {
	Statistics EssData8
	Direction  struct {
		IsGridSelling        int `json:"is_grid_selling_,string"`
		IsBatteryDischarging int `json:"is_battery_discharging_,string"`
	}
}

func (m MeterResponse8) GetGridPower() float64 {
	if m.Direction.IsGridSelling > 0 {
		return -m.Statistics.GridPower
	}
	return m.Statistics.GridPower
}

func (m MeterResponse8) GetPvTotalPower() float64 {
	return m.Statistics.PvTotalPower
}

func (m MeterResponse8) GetBatConvPower() float64 {
	// discharge battery: batPower is positive, charge battery: batPower is negative
	if m.Direction.IsBatteryDischarging == 0 {
		return -m.Statistics.BatConvPower
	}
	return m.Statistics.BatConvPower
}

func (m MeterResponse8) GetBatUserSoc() float64 {
	return m.Statistics.BatUserSoc
}

func (m MeterResponse8) GetCurrentGridFeedInEnergy() float64 {
	return m.Statistics.CurrentGridFeedInEnergy
}

func (m MeterResponse8) GetCurrentPvGenerationSum() float64 {
	return m.Statistics.CurrentPvGenerationSum
}

// power values are in 100W units
type EssData15 struct {
	GridPower    int `json:"grid_power_01kW"`
	PvTotalPower int `json:"pv_total_power_01kW"`
	BatConvPower int `json:"batt_conv_power_01kW"`
	BatUserSoc   int `json:"bat_user_soc"`
}

type MeterResponse15 struct {
	Statistics EssData15
	Direction  struct {
		IsGridSelling        int `json:"is_grid_selling_"`
		IsBatteryDischarging int `json:"is_battery_discharging_"`
	}
}

func (m MeterResponse15) GetGridPower() float64 {
	if m.Direction.IsGridSelling > 0 {
		return -float64(m.Statistics.GridPower * 100)
	}
	return float64(m.Statistics.GridPower * 100)
}

func (m MeterResponse15) GetPvTotalPower() float64 {
	return float64(m.Statistics.PvTotalPower * 100)
}

func (m MeterResponse15) GetBatConvPower() float64 {
	// discharge battery: batPower is positive, charge battery: batPower is negative
	if m.Direction.IsBatteryDischarging == 0 {
		return -float64(m.Statistics.BatConvPower * 100)
	}
	return float64(m.Statistics.BatConvPower * 100)
}

func (m MeterResponse15) GetBatUserSoc() float64 {
	return float64(m.Statistics.BatUserSoc)
}

func (m MeterResponse15) GetCurrentGridFeedInEnergy() float64 {
	return math.NaN() // data not provided by Ess15
}

func (m MeterResponse15) GetCurrentPvGenerationSum() float64 {
	return math.NaN() // data not provided by Ess15
}
