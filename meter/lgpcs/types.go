package lgpcs

type MeterResponse8 struct {
	Statistics EssData8
	Direction  struct {
		IsGridSelling        int `json:"is_grid_selling_,string"`
		IsBatteryDischarging int `json:"is_battery_discharging_,string"`
	}
}

type EssData8 struct {
	GridPower               float64 `json:"grid_power,string"`
	PvTotalPower            float64 `json:"pcs_pv_total_power,string"`
	BatConvPower            float64 `json:"batconv_power,string"`
	BatUserSoc              float64 `json:"bat_user_soc,string"`
	CurrentGridFeedInEnergy float64 `json:"current_grid_feed_in_energy,string"`
	CurrentPvGenerationSum  float64 `json:"current_pv_generation_sum,string"`
}

type MeterResponse15 struct {
	Statistics EssData15
	Direction  struct {
		IsGridSelling        int `json:"is_grid_selling_"`
		IsBatteryDischarging int `json:"is_battery_discharging_"`
	}
}

type EssData15 struct {
	GridPower    int `json:"grid_power_01kW"`
	PvTotalPower int `json:"pv_total_power_01kW"`
	BatConvPower int `json:"batt_conv_power_01kW"`
	BatUserSoc   int `json:"bat_user_soc"`
}
