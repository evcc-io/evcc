package shelly

// Gen2API endpoint reference: https://shelly-api-docs.shelly.cloud/gen2/

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
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

type Gen2EMData struct {
	TotalAct    float64 `json:"total_act"`
	TotalActRet float64 `json:"total_act_ret"`
}

type Gen2EM1Status struct {
	Current  float64 `json:"current"`
	Voltage  float64 `json:"voltage"`
	ActPower float64 `json:"act_power"`
}

type Gen2EM1Data struct {
	TotalActEnergy    float64 `json:"total_act_energy"`
	TotalActRetEnergy float64 `json:"total_act_ret_energy"`
}

type Gen2StatusResponse struct {
	Switch0 Gen2SwitchStatus `json:"switch:0"`
	Switch1 Gen2SwitchStatus `json:"switch:1"`
	Switch2 Gen2SwitchStatus `json:"switch:2"`
	Pm0     Gen2SwitchStatus `json:"pm1:0"`
	Pm1     Gen2SwitchStatus `json:"pm2:1"`
	Pm2     Gen2SwitchStatus `json:"pm3:2"`
	// additional shelly Pro EM meter JSON response
	TotalPower float64       `json:"total_act_power"`
	CurrentA   float64       `json:"a_current"`
	CurrentB   float64       `json:"b_current"`
	CurrentC   float64       `json:"c_current"`
	VoltageA   float64       `json:"a_voltage"`
	VoltageB   float64       `json:"b_voltage"`
	VoltageC   float64       `json:"c_voltage"`
	PowerA     float64       `json:"a_act_power"`
	PowerB     float64       `json:"b_act_power"`
	PowerC     float64       `json:"c_act_power"`
	Em0        Gen2EM1Status `json:"em1:0"`
	Em1        Gen2EM1Status `json:"em1:1"`
	Em2        Gen2EM1Status `json:"em1:2"`
	Em0Data    Gen2EM1Data   `json:"em1data:0"`
	Em1Data    Gen2EM1Data   `json:"em1data:1"`
	Em2Data    Gen2EM1Data   `json:"em1data:2"`
}

type Gen2EmDataStatusResponse struct {
	TotalEnergy float64 `json:"total_act"`
}
