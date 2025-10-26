package marstekvenusapi

import (
	"encoding/json"
)

const (
	// JSON-RPC Error Codes (from API spec)
	ERROR_PARSE_ERROR      = -32700
	ERROR_INVALID_REQUEST  = -32600
	ERROR_METHOD_NOT_FOUND = -32601
	ERROR_INVALID_PARAMS   = -32602
	ERROR_INTERNAL_ERROR   = -32603

	// API Methods
	METHOD_GET_DEVICE     = "Marstek.GetDevice"
	METHOD_WIFI_STATUS    = "Wifi.GetStatus"
	METHOD_BLE_STATUS     = "BLE.GetStatus"
	METHOD_BATTERY_STATUS = "Bat.GetStatus"
	METHOD_PV_STATUS      = "PV.GetStatus"
	METHOD_ES_STATUS      = "ES.GetStatus"
	METHOD_ES_MODE        = "ES.GetMode"
	METHOD_ES_SET_MODE    = "ES.SetMode"
	METHOD_EM_STATUS      = "EM.GetStatus"

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

type RequestType string

type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    int    `json:"data"`
}

// Response Wrapper
type Response struct {
	ID     int             `json:"id"`
	Src    string          `json:"src"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorInfo      `json:"error,omitempty"`
}

type RpcRequest struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
}

// ---------------------------------------------------
type GetStatusReqParams struct {
	Id int `json:"id"`
}

// ---------------------------------------------------
type BatGetStatusReq struct {
	RpcRequest
	Params *GetStatusReqParams `json:"params"`
}

type BatStatusResult struct {
	ID                 int     `json:"id"`
	BatSoc             int     `json:"soc"`
	ChargingAllowed    bool    `json:"charg_flag"`
	DischargingAllowed bool    `json:"dischrg_flag"`
	BatTemp            float64 `json:"bat_temp"`
	BatteryCapacity    float64 `json:"bat_capacity"`
	RatedCapacity      float64 `json:"rated:capacity"`
}

// ---------------------------------------------------
type GetDeviceReqParams struct {
	BleMac string `json:"ble_mac"`
}

type GetDeviceReq struct {
	RpcRequest
	Params *GetDeviceReqParams `json:"params"`
}

type DeviceResult struct {
	Device   string `json:"device"`
	Version  int    `json:"ver"`
	BleMac   string `json:"ble_mac"`
	WifiMac  string `json:"wifi_mac"`
	WifiName string `json:"wifi_name"`
	IpAddr   string `json:"ip"`
}

// ---------------------------------------------------
type EsGetStatusReq struct {
	RpcRequest
	Params *GetStatusReqParams `json:"params"`
}

type EsStatusResult struct {
	Id                    int `json:"id"`
	BatSoc                int `json:"bat_soc"`
	BatCap                int `json:"bat_cap"`
	PvPower               int `json:"pv_power"`
	OnGridPower           int `json:"ongrid_power"`
	OffGridPower          int `json:"offgrid_power"`
	BatPower              int `json:"bat_power"`
	TotalPvEnerv          int `json:"total_pv_energy"`
	TotalGridOutputEnergy int `json:"total_grid_output_energy"`
	TotalGridInputEnergy  int `json:"total_grid_input_energy"`
	TotalLoadEnergy       int `json:"total_load_energy"`
}

// ---------------------------------------------------
type EsGetModeReq struct {
	RpcRequest
	Params *GetStatusReqParams `json:"params"`
}

type EsModeResult struct {
	Id           int    `json:"id"`
	Mode         string `json:"mode"` // "mode": "Passive", "Manual", "AI", "Auto"
	OnGridPower  int    `json:"ongrid_power"`
	OffGridPower int    `json:"offgrid_power"`
	BatSoc       int    `json:"bat_soc"`
}

// ---------------------------------------------------
type MRAutoConfig struct {
	Enable int `json:"enable"`
}

type MRAIConfig struct {
	Enable int `json:"enable"`
}

type MRManualConfig struct {
	TimeNum   int    `json:"time_num"`   //Time period serial number, Venus C/E supports 0-9
	StartTime string `json:"start_time"` //Start time, hours: minutes, [hh:mm]
	EndTime   string `json:"end_time"`   // End time, hours: minutes, [hh:mm]

	WeekSet int `json:"week_set"`
	// Week, a byte 8 bits, the low 7 bits effective, the highest
	// invalid, 0000 0001 (1) on behalf of Monday open, 0000 0011
	// (3) on behalf of Monday and Tuesday open, 0111 1111 (127)
	// on behalf of a week
	Power  int `json:"power"`  //Setting power,[W]
	Enable int `json:"enable"` //ON: 1; OFF: 0
}

type MRPassiveConfig struct {
	Power  int `json:"power"`   // Setting power,[W]
	CdTime int `json:"cd_time"` // Power countdown,[s]
}

type ModeReqConfig struct {
	ConfigMode    string `json:"mode"`
	AutoConfig    *MRAutoConfig
	AIConfig      *MRAIConfig
	ManualConfig  *MRManualConfig
	PassiveConfig *MRPassiveConfig
}

type SetModeReqParams struct {
	Id     int `json:"id"`
	Config *ModeReqConfig
}

type EsSetModeReq struct {
	RpcRequest
	Params *SetModeReqParams `json:"params"`
}

type EsSetModeResult struct {
	Id          int  `json:"id"`
	SetResultOk bool `json:"set_result,string"`
}

// ---------------------------------------------------
