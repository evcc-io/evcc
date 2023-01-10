package shelly

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
}

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1StatusResponse struct {
	Meters []struct {
		Power float64
		Total int64
	}
	// Shelly EM meter JSON response
	EMeters []struct {
		Power float64
		Total int64
	}
}
