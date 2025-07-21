package optimizer

type Request struct {
	Ems                   Ems        `json:"ems"`
	Inverter              Inverter   `json:"inverter"`
	PvAkku                PvAkku     `json:"pv_akku"`
	EAuto                 EAuto      `json:"eauto"`
	TemperaturePrediction Prediction `json:"temperature_prediction"`
}

type Ems struct {
	PreisEuroProWhAkku           float64    `json:"preis_euro_pro_wh_akku"`
	EinspeiseverguetungEuroProWh Prediction `json:"einspeiseverguetung_euro_pro_wh"`
	Gesamtlast                   Prediction `json:"gesamtlast"`
	PvPrognoseWh                 Prediction `json:"pv_prognose_wh"`
	StrompreisEuroProWh          Prediction `json:"strompreis_euro_pro_wh"`
}

type EAuto struct {
	DeviceID              string  `json:"device_id"`
	CapacityWh            int     `json:"capacity_wh"`
	ChargingEfficiency    float64 `json:"charging_efficiency,omitempty"`
	DischargingEfficiency float64 `json:"discharging_efficiency,omitempty"`
	MaxChargePowerW       int     `json:"max_charge_power_w"`
	InitialSocPercentage  int     `json:"initial_soc_percentage"`
	MinSocPercentage      int     `json:"min_soc_percentage"`
}

type PvAkku struct {
	DeviceID             string `json:"device_id"`
	CapacityWh           int    `json:"capacity_wh"`
	MaxChargePowerW      int    `json:"max_charge_power_w"`
	InitialSocPercentage int    `json:"initial_soc_percentage"`
	MinSocPercentage     int    `json:"min_soc_percentage,omitempty"`
}

type Inverter struct {
	DeviceID   string `json:"device_id"`
	BatteryID  string `json:"battery_id"`
	MaxPowerWh int    `json:"max_power_wh"`
}

type Prediction [48]float64

func (f Prediction) Div(factor float64) Prediction {
	var res Prediction
	for i, v := range f {
		res[i] = v / factor
	}
	return res
}

type Response struct {
	LastWhProStunde            Forecast `json:"Last_Wh_pro_Stunde"`        // of hourly load values in Wh
	EAutoSocProStunde          Forecast `json:"EAuto_SoC_pro_Stunde"`      // of hourly EV state of charge values (%)
	EinnahmenEuroProStunde     Forecast `json:"Einnahmen_Euro_pro_Stunde"` // of hourly revenue values in Euro
	AkkuSocProStunde           Forecast `json:"akku_soc_pro_stunde"`       // of hourly battery state of charge values (%)
	GesamtVerluste             float64  `json:"Gesamt_Verluste"`           // energy losses in Wh
	GesamtbilanzEuro           float64  `json:"Gesamtbilanz_Euro"`         // financial balance in Euro
	Gesamteinnahmen_Euro       float64  `json:"Gesamteinnahmen_Euro"`      // revenue in Euro
	GesamtkostenEuro           float64  `json:"Gesamtkosten_Euro"`         // costs in Euro
	NetzbezugWhProStunde       Forecast `json:"Netzbezug_Wh_pro_Stunde"`
	NetzeinspeisungWhProStunde Forecast `json:"Netzeinspeisung_Wh_pro_Stunde"`
}

type Forecast []float64
