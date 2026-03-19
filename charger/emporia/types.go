package emporia

const (
	// API base URL
	BaseURL = "https://api.emporiaenergy.com"

	// API endpoints
	DevicesStatusEndpoint = BaseURL + "/customers/devices/status"
	ChargerEndpoint       = BaseURL + "/devices/evcharger"

	// AWS Cognito configuration for Emporia
	AWSRegion = "us-east-2"
	ClientID  = "4qte47jbstod8apnfic0bunmrq"
	UserPool  = "us-east-2_ghlOXVLi1"
)

// DevicesStatus is the response from the devices status endpoint
type DevicesStatus struct {
	EvChargers []ChargerDevice `json:"evChargers"`
}

// ChargerDevice represents an Emporia EVSE device
type ChargerDevice struct {
	DeviceGid               int64   `json:"deviceGid"`
	LoadGid                 int64   `json:"loadGid"`
	Message                 string  `json:"message"`
	Status                  string  `json:"status"`
	Icon                    string  `json:"icon"`
	IconLabel               string  `json:"iconLabel"`
	IconDetailText          string  `json:"iconDetailText"`
	FaultText               *string `json:"faultText"`
	ChargerOn               bool    `json:"chargerOn"`
	ChargingRate            int     `json:"chargingRate"`
	MaxChargingRate         int     `json:"maxChargingRate"`
	OffPeakSchedulesEnabled bool    `json:"offPeakSchedulesEnabled"`
}

// ChargerUpdate is the request body for updating a charger
type ChargerUpdate struct {
	DeviceGid       int64 `json:"deviceGid"`
	LoadGid         int64 `json:"loadGid"`
	ChargerOn       bool  `json:"chargerOn"`
	ChargingRate    int   `json:"chargingRate"`
	MaxChargingRate int   `json:"maxChargingRate"`
}
