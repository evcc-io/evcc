package chargepoint

// HomeChargerStatus holds the current status of a home charger.
type HomeChargerStatus struct {
	ChargingStatus         string `json:"chargingStatus"` // AVAILABLE, CHARGING
	IsPluggedIn            bool   `json:"isPluggedIn"`    // Vehicle's connection status.
	IsConnected            bool   `json:"isConnected"`    // ???
	ChargeAmperageSettings struct {
		ChargeLimit         int64   `json:"chargeLimit"`         // What the limit is now.
		PossibleChargeLimit []int64 `json:"possibleChargeLimit"` // Possible limits.
	} `json:"chargeAmperageSettings"`
}

// DeviceData is the iOS device fingerprint included in ChargePoint API requests.
type DeviceData struct {
	AppID              string `json:"appId"`
	Manufacturer       string `json:"manufacturer"`
	Model              string `json:"model"`
	NotificationID     string `json:"notificationId"`
	NotificationIDType string `json:"notificationIdType"`
	Type               string `json:"type"`
	UDID               string `json:"udid"`
	Version            string `json:"version"`
}

type endpointValue struct {
	Value string `json:"value"`
}

type configEndpoints struct {
	Accounts    endpointValue `json:"accounts_endpoint"`
	Chargers    endpointValue `json:"hcpo_hcm_endpoint"`
	InternalAPI endpointValue `json:"internal_api_gateway_endpoint"`
	MapCache    endpointValue `json:"mapcache_endpoint"`
	SSO         endpointValue `json:"sso_endpoint"`
	WebServices endpointValue `json:"webservices_endpoint"`
}

type globalConfig struct {
	Region    string          `json:"region"`
	EndPoints configEndpoints `json:"endPoints"`
}

type accountLoginResponse struct {
	SessionID    string `json:"sessionId"`
	SSOSessionID string `json:"ssoSessionId"`
	User         struct {
		UserID int32 `json:"userId"`
	} `json:"user"`
}
