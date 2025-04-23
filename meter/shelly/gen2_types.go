package shelly

// Gen2API endpoint reference: https://shelly-api-docs.shelly.cloud/gen2/

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
}

type Gen2Methods struct {
	Methods []string
}

type Gen2SwitchStatus struct {
	Output  bool
	Apower  float64
	Voltage float64
	Current float64
	Aenergy struct {
		Total float64
	}
	Ret_Aenergy struct {
		Total float64
	}
}

var _ Phases = (*Gen2EMStatus)(nil)

type Gen2EMStatus struct {
	TotalActPower float64 `json:"total_act_power"`
	ACurrent      float64 `json:"a_current"`
	BCurrent      float64 `json:"b_current"`
	CCurrent      float64 `json:"c_current"`
	AVoltage      float64 `json:"a_voltage"`
	BVoltage      float64 `json:"b_voltage"`
	CVoltage      float64 `json:"c_voltage"`
	AActPower     float64 `json:"a_act_power"`
	BActPower     float64 `json:"b_act_power"`
	CActPower     float64 `json:"c_act_power"`
}

// Currents implements the api.PhaseCurrents interface
func (res Gen2EMStatus) Currents() (float64, float64, float64, error) {
	return res.ACurrent, res.BCurrent, res.CCurrent, nil
}

// Voltages implements the api.PhaseVoltages interface
func (res Gen2EMStatus) Voltages() (float64, float64, float64, error) {
	return res.AVoltage, res.BVoltage, res.CVoltage, nil
}

// Powers implements the api.PhasePowers interface
func (res Gen2EMStatus) Powers() (float64, float64, float64, error) {
	return res.AActPower, res.BActPower, res.CActPower, nil
}

type Gen2EMData struct {
	TotalAct    float64 `json:"total_act"`
	TotalActRet float64 `json:"total_act_ret"`
}

var _ Phases = (*Gen2EM1Status)(nil)

type Gen2EM1Status struct {
	Current  float64 `json:"current"`
	Voltage  float64 `json:"voltage"`
	ActPower float64 `json:"act_power"`
}

// Currents implements the api.PhaseCurrents interface
func (res Gen2EM1Status) Currents() (float64, float64, float64, error) {
	return res.Current, 0, 0, nil
}

// Voltages implements the api.PhaseVoltages interface
func (res Gen2EM1Status) Voltages() (float64, float64, float64, error) {
	return res.Voltage, 0, 0, nil
}

// Powers implements the api.PhasePowers interface
func (res Gen2EM1Status) Powers() (float64, float64, float64, error) {
	return res.ActPower, 0, 0, nil
}

type Gen2EM1Data struct {
	TotalActEnergy    float64 `json:"total_act_energy"`
	TotalActRetEnergy float64 `json:"total_act_ret_energy"`
}
