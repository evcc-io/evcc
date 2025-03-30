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

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1StatusResponse struct {
	Meters []struct {
		Power          float64
		Total          float64
		Total_Returned float64
	}
	// Shelly EM meter JSON response
	EMeters []struct {
		Power          float64
		Total          float64
		Total_Returned float64
	}
}
