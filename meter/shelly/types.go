package shelly

// DeviceInfo is the common /shelly endpoint response
// https://shelly-api-docs.shelly.cloud/gen1/#shelly
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Shelly#http-endpoint-shelly
type DeviceInfo struct {
	Mac       string `json:"mac"`
	Gen       int    `json:"gen"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	Auth      bool   `json:"auth"`
	AuthEn    bool   `json:"auth_en"`
	NumMeters int    `json:"num_meters"`
	Profile   string `json:"profile"`
}
