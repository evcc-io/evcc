package wallbox

const ApiURI = "https://api.wall-box.com"

type Token struct {
	Jwt string
}

type Charger struct {
	ID                   int
	UID                  string
	Image                string
	MaxChargingCurrent   int     // null,
	AddedEnergy          float64 // 9.672,
	ChargingPower        float64 // 0,
	ChargingTime         int     // 9370,
	State                int
	OperationMode        string
	OcppConnectionStatus int
	MidStatus            int
	Locked               int
	RemoteAction         int
}

type Group struct {
	Chargers []Charger
}

type Groups struct {
	Result struct {
		Groups []Group
	}
}

type Error struct {
	Code int
	Msg  string
}

type ChargerStatus struct {
	Error
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	// CarID                 int     `json:"car_id"`
	// CarPlate              string  `json:"car_plate"`
	// DepotPrice            float64 `json:"depot_price"`
	// LastSync              string  `json:"last_sync"`
	// PowerSharingStatus    int     `json:"power_sharing_status"`
	// MidStatus             int     `json:"mid_status"`
	StatusID int `json:"status_id"`
	// Name                  string  `json:"name"`
	// ChargingPower         float64 `json:"charging_power"`
	// MaxAvailablePower     float64 `json:"max_available_power"`
	// DepotName             string  `json:"depot_name"`
	// ChargingSpeed         int     `json:"charging_speed"`
	// AddedRange            int     `json:"added_range"`
	// AddedEnergy           float64 `json:"added_energy"`
	// AddedGreenEnergy      float64 `json:"added_green_energy"`
	// AddedDischargedEnergy float64 `json:"added_discharged_energy"`
	// ChargingTime          int     `json:"charging_time"`
	// Finished              bool    `json:"finished"`
	// Cost                  float64 `json:"cost"`
	CurrentMode int `json:"current_mode"` // phases
	// PreventiveDischarge   bool    `json:"preventive_discharge"`
	// StateOfCharge         int     `json:"state_of_charge"`
	// OcppStatus            int     `json:"ocpp_status"`
	ConfigData struct {
		ChargerID int `json:"charger_id"`
		// Uid                string  `json:"uid"`
		// SerialNumber       string  `json:"serial_number"`
		// Name               string  `json:"name"`
		// Locked             int     `json:"locked"`
		// AutoLock           int     `json:"auto_lock"`
		// AutoLockTime       int     `json:"auto_lock_time"`
		// Multiuser          int     `json:"multiuser"`
		MaxChargingCurrent int `json:"max_charging_current"`
		// Language           string  `json:"language"`
		// IcpMaxCurrent      int     `json:"icp_max_current"`
		// GridType           int     `json:"grid_type"`
		// EnergyPrice        float64 `json:"energy_price"`
		// EnergyCost         struct {
		// 	Value            float64 `json:"value"`
		// 	InheritedGroupId int     `json:"inheritedGroupId"`
		// } `json:"energyCost"`
		// UnlockUserID        int     `json:"unlock_user_id"`
		// PowerSharingConfig  int     `json:"power_sharing_config"`
		// PurchasedPower      float64 `json:"purchased_power"`
		// ShowName            int     `json:"show_name"`
		// ShowLastname        int     `json:"show_lastname"`
		// ShowEmail           int     `json:"show_email"`
		// ShowProfile         int     `json:"show_profile"`
		// ShowDefaultUser     int     `json:"show_default_user"`
		// GestureStatus       int     `json:"gesture_status"`
		// HomeSharing         int     `json:"home_sharing"`
		// DcaStatus           int     `json:"dca_status"`
		// ConnectionType      int     `json:"connection_type"`
		// MaxAvailableCurrent int     `json:"max_available_current"`
		// LiveRefreshTime     int     `json:"live_refresh_time"`
		// UpdateRefreshTime   int     `json:"update_refresh_time"`
		// OwnerID             int     `json:"owner_id"`
		// RemoteAction        int     `json:"remote_action"`
		// RfidType            int     `json:"rfid_type"`
		// ChargerHasImage     int     `json:"charger_has_image"`
		// Sha256ChargerImage  string  `json:"sha256_charger_image"`
		// Plan                struct {
		// 	PlanName string      `json:"plan_name"`
		// 	Features interface{} `json:"features"`
		// } `json:"plan"`
		// SyncTimestamp int `json:"sync_timestamp"`
		// Currency      struct {
		// 	Id     int    `json:"id"`
		// 	Name   string `json:"name"`
		// 	Symbol string `json:"symbol"`
		// 	Code   string `json:"code"`
		// } `json:"currency"`
		// ChargerLoadType           string `json:"charger_load_type"`
		// ContractChargingAvailable bool   `json:"contract_charging_available"`
		// Country                   struct {
		// 	Id         int    `json:"id"`
		// 	Code       string `json:"code"`
		// 	Iso2       string `json:"iso2"`
		// 	Name       string `json:"name"`
		// 	Phone_code string `json:"phone_code"`
		// } `json:"country"`
		// State      int    `json:"state"`
		// Timezone   string `json:"timezone"`
		// PartNumber string `json:"part_number"`
		// Software   struct {
		// 	UpdateAvailable bool   `json:"updateAvailable"`
		// 	CurrentVersion  string `json:"currentVersion"`
		// 	LatestVersion   string `json:"latestVersion"`
		// } `json:"software"`
		// Available            int         `json:"available"`
		// OperationMode        string      `json:"operation_mode"`
		// OcppReady            string      `json:"ocpp_ready"`
		// Tariffs              interface{} `json:"tariffs"`
		// MidEnabled           int         `json:"mid_enabled"`
		// MidMargin            int         `json:"mid_margin"`
		// MidMarginUnit        int         `json:"mid_margin_unit"`
		// MidSerialNumber      string      `json:"mid_serial_number"`
		// MidStatus            int         `json:"mid_status"`
		// SessionSegmentLength int         `json:"session_segment_length"`
		// GroupID              int         `json:"group_id"`
	} `json:"config_data"`
}

func (s ChargerStatus) Status() Status {
	switch s.StatusID {
	case 164, 180, 181, 183, 184, 185, 186, 187, 188, 189:
		return WAITING
	case 193, 194, 195:
		return CHARGING
	case 161, 162:
		return READY
	case 178, 182:
		return PAUSED
	case 177, 179:
		return SCHEDULED
	case 196:
		return DISCHARGING
	case 14, 15:
		return ERROR
	case 0, 163:
		return DISCONNECTED
	case 209, 210, 165:
		return LOCKED
	case 166:
		return UPDATING
	default:
		return ERROR
	}
}
