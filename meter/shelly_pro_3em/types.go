package shelly_pro_3em

// Shelly api homepage
// https://shelly-api-docs.shelly.cloud/#common-http-api
type DeviceInfo struct {
	Gen       int    `json:"gen"`
	Id        string `json:"id"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	Mac       string `json:"mac"`
	Auth      bool   `json:"auth"`
	AuthEn    bool   `json:"auth_en"`
	NumMeters int    `json:"num_meters"`
}

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
}

type Gen2EmStatusResponse struct {
	/*
	"id": 0,
  "a_current": 0.987,
  "a_voltage": 229.1,
  "a_act_power": 173,
  "a_aprt_power": 226.1,
  "a_pf": -0.81,
  "b_current": 0.415,
  "b_voltage": 230.1,
  "b_act_power": 52.1,
  "b_aprt_power": 95.6,
  "b_pf": -0.69,
  "c_current": 0.954,
  "c_voltage": 230.4,
  "c_act_power": 181.1,
  "c_aprt_power": 219.7,
  "c_pf": -0.85,
  "n_current": null,
  "total_current": 2.357,
  "total_act_power": 406.215,
  "total_aprt_power": 541.334*/
  TotalPower 	float64 	`json:"total_act_power"`
  CurrentA 		float64 	`json:"a_current"`
  CurrentB 		float64 	`json:"b_current"`
  CurrentC 		float64 	`json:"c_current"`
  VoltageA 		float64 	`json:"a_voltage"`
  VoltageB 		float64 	`json:"b_voltage"`
  VoltageC 		float64 	`json:"c_voltage"`
  PowerA 		float64 	`json:"a_act_power"`
  PowerB 		float64 	`json:"b_act_power"`
  PowerC 		float64 	`json:"c_act_power"`
}

type Gen2EmDataStatusResponce struct {
	/*
	"id": 0,
  "a_total_act_energy": 1491.82,
  "a_total_act_ret_energy": 0,
  "b_total_act_energy": 2456.74,
  "b_total_act_ret_energy": 0,
  "c_total_act_energy": 1175.25,
  "c_total_act_ret_energy": 0,
  "total_act": 5123.81,
  "total_act_ret": 0*/
  TotalEnergy 	float64 	`json:"total_act"`
}
