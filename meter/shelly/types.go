package shelly

// Shelly api homepage
// https://shelly-api-docs.shelly.cloud/#common-http-api
type DeviceInfo struct {
	Gen       int    `json:"gen"`
	Id        string `json:"id"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	Mac       string `json:"mac"`
	App       string `json:"app"`
	Auth      bool   `json:"auth"`
	AuthEn    bool   `json:"auth_en"`
	NumMeters int    `json:"num_meters"`
	Profile   string `json:"profile"`
}

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
}

type Gen2SwitchResponse struct {
	Output bool `json:"output"`
}

type Gen2Switch struct {
	Apower  float64
	Aenergy struct {
		Total float64
	}
}

type Gen2StatusResponse struct {
	Switch0 Gen2Switch `json:"switch:0"`
	Switch1 Gen2Switch `json:"switch:1"`
	Switch2 Gen2Switch `json:"switch:2"`
	Pm0     Gen2Switch `json:"pm1:0"`
	Pm1     Gen2Switch `json:"pm2:1"`
	Pm2     Gen2Switch `json:"pm3:2"`
}

type Gen2Em struct {
	Current  float64 `json:"current"`
	Voltage  float64 `json:"voltage"`
	ActPower float64 `json:"act_power"`
}

type Gen2EmData struct {
	TotalActEnergy    float64 `json:"total_act_energy"`
	TotalActRetEnergy float64 `json:"total_act_ret_energy"`
}

type Gen2EmStatusResponse struct {
	TotalPower float64    `json:"total_act_power"`
	CurrentA   float64    `json:"a_current"`
	CurrentB   float64    `json:"b_current"`
	CurrentC   float64    `json:"c_current"`
	VoltageA   float64    `json:"a_voltage"`
	VoltageB   float64    `json:"b_voltage"`
	VoltageC   float64    `json:"c_voltage"`
	PowerA     float64    `json:"a_act_power"`
	PowerB     float64    `json:"b_act_power"`
	PowerC     float64    `json:"c_act_power"`
	Em0        Gen2Em     `json:"em1:0"`
	Em1        Gen2Em     `json:"em1:1"`
	Em2        Gen2Em     `json:"em1:2"`
	Em0Data    Gen2EmData `json:"em1data:0"`
	Em1Data    Gen2EmData `json:"em1data:1"`
	Em2Data    Gen2EmData `json:"em1data:2"`
}

type Gen2EmDataStatusResponse struct {
	TotalEnergy float64 `json:"total_act"`
}

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1StatusResponse struct {
	Meters []struct {
		Power float64
		Total float64
	}
	// Shelly EM meter JSON response
	EMeters []struct {
		Power float64
		Total float64
	}
}
