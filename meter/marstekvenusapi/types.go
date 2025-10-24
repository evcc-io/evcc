package marstekvenusapi

const (
	// JSON-RPC Error Codes (from API spec)
	ERROR_PARSE_ERROR      = -32700
	ERROR_INVALID_REQUEST  = -32600
	ERROR_METHOD_NOT_FOUND = -32601
	ERROR_INVALID_PARAMS   = -32602
	ERROR_INTERNAL_ERROR   = -32603

	// API Methods
	// METHOD_GET_DEVICE = "Marstek.GetDevice"
	// METHOD_WIFI_STATUS = "Wifi.GetStatus"
	// METHOD_BLE_STATUS = "BLE.GetStatus"
	METHOD_BATTERY_STATUS = "Bat.GetStatus"
	// METHOD_PV_STATUS = "PV.GetStatus"
	METHOD_ES_STATUS   = "ES.GetStatus"
	METHOD_ES_MODE     = "ES.GetMode"
	METHOD_ES_SET_MODE = "ES.SetMode"
	// METHOD_EM_STATUS = "EM.GetStatus"

	// Operating modes
	MODE_AUTO    = "Auto"
	MODE_AI      = "AI"
	MODE_MANUAL  = "Manual"
	MODE_PASSIVE = "Passive"

	// Battery states
	BATTERY_STATE_IDLE        = "idle"
	BATTERY_STATE_CHARGING    = "charging"
	BATTERY_STATE_DISCHARGING = "discharging"

	DEFAULT_PORT = 30000
)

// Report contains report id and device serial
type Response struct {
	ID     int    `json:"ID,string"`
	Serial string `json:"Serial"`
}

type Bat_GetStatus struct {
	StateOfCharge      float64 `json:"soc"`
	ChargingAllowed    bool    `json:"charg_flag"`
	DischargingAllowed bool    `json:"dischrg_flag"`
	BatTemp            float64 `json:"bat_temp"`
	BatteryCapacity    float64 `json:"bat_capacity"`
	RatedCapacity      float64 `json:"rated:capacity"`
}
