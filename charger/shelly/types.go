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

type Gen2SwitchResponse struct {
	Output bool `json:"output,omitempty"`
}

type Switch struct {
	Apower float64
}

type Gen2StatusResponse struct {
	Switch0 Switch `json:"switch:0"`
	Switch1 Switch `json:"switch:1"`
	Switch2 Switch `json:"switch:2"`
}

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1StatusResponse struct {
	Meters []struct {
		Power float64
	}
}
