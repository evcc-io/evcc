package shelly

// Shelly api homepage
// https://shelly-api-docs.shelly.cloud/#common-http-api
type DeviceInfo struct {
	Gen       int    `json:"gen,omitempty"`
	Id        string `json:"id,omitempty"`
	Model     string `json:"model,omitempty"`
	Type      string `json:"type,omitempty"`
	Mac       string `json:"mac,omitempty"`
	Auth      bool   `json:"auth,omitempty"`
	AuthEn    bool   `json:"auth_en,omitempty"`
	NumMeters int    `json:"num_meters,omitempty"`
}

type Gen2SwitchResponse struct {
	Output bool `json:"output,omitempty"`
}

type Gen2StatusResponse struct {
	Switch0 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:0,omitempty"`
	Switch1 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:1,omitempty"`
	Switch2 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:2,omitempty"`
}

type Gen1SwitchResponse struct {
	Ison bool `json:"ison,omitempty"`
}

type Gen1StatusResponse struct {
	Meters []struct {
		Power float64 `json:"power,omitempty"`
	} `json:"meters,omitempty"`
}
