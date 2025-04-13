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

type SystemInfoResponse struct {
	Battery BatteryInfo `json:"batt"`
	PMS     PMS         `json:"pms"`
	Version Version     `json:"version"`
}

type BatteryInfo struct {
	Capacity          int    `json:"capacity,string"`
	HBCAPackDates     string `json:"hbc_a_pack_dates"`
	HBCASerials       string `json:"hbc_a_serials"`
	HBCBPackDates     string `json:"hbc_b_pack_dates"`
	HBCBSerials       string `json:"hbc_b_serials"`
	HBCChgCap1        int    `json:"hbc_chg_cap_1,string"`
	HBCChgCap2        int    `json:"hbc_chg_cap_2,string"`
	HBCChgEnergy1     int    `json:"hbc_chg_energy_1,string"`
	HBCChgEnergy2     int    `json:"hbc_chg_energy_2,string"`
	HBCCycleCount1    int    `json:"hbc_cycle_count_1,string"`
	HBCCycleCount2    int    `json:"hbc_cycle_count_2,string"`
	HBCDeepDischgCnt1 int    `json:"hbc_deep_dischg_cnt_1,string"`
	HBCDeepDischgCnt2 int    `json:"hbc_deep_dischg_cnt_2,string"`
	HBCDischgCap1     int    `json:"hbc_dischg_cap_1,string"`
	HBCDischgCap2     int    `json:"hbc_dischg_cap_2,string"`
	HBCDischgEnergy1  int    `json:"hbc_dischg_energy_1,string"`
	HBCDischgEnergy2  int    `json:"hbc_dischg_energy_2,string"`
	HBCDischgRate1    int    `json:"hbc_dischg_rate_1,string"`
	HBCDischgRate2    int    `json:"hbc_dischg_rate_2,string"`
	HBCOVerChgCnt1    int    `json:"hbc_over_chg_cnt_1,string"`
	HBCOVerChgCnt2    int    `json:"hbc_over_chg_cnt_2,string"`
	HBCRemainingCap1  int    `json:"hbc_remaining_cap_1,string"`
	HBCRemainingCap2  int    `json:"hbc_remaining_cap_2,string"`
	InstallDate       string `json:"install_date"`
	NameplateEnergy1  int    `json:"nameplate_energy_1,string"`
	NameplateEnergy2  int    `json:"nameplate_energy_2,string"`
	BatteryType       string `json:"type"`
}

type PMS struct {
	ACInputPower  int    `json:"ac_input_power,string"`
	ACOutputPower int    `json:"ac_output_power,string"`
	InstallDate   string `json:"install_date"`
	Model         string `json:"model"`
	SerialNo      string `json:"serialno"`
}

type Version struct {
	BMSUnit1Version string `json:"bms_unit1_version"`
	BMSUnit2Version string `json:"bms_unit2_version"`
	BMSVersion      string `json:"bms_version"`
	PCSVersion      string `json:"pcs_version"`
	PMSBuildDate    string `json:"pms_build_date"`
	PMSVersion      string `json:"pms_version"`
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
