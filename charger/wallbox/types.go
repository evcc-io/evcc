package wallbox

const ApiURI = "https://api.wall-box.com"

type Token struct {
	Jwt string
}

type Groups struct {
	Result struct {
		Groups []struct {
			Chargers []struct {
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
		}
	}
}

type ChargerStatus struct {
	UserID                int     `json:"user_id"`              // 1
	UserName              string  `json:"user_name"`            // "default"
	CarID                 int     `json:"car_id"`               // 1
	CarPlate              string  `json:"car_plate"`            // ""
	DepotPrice            float64 `json:"depot_price"`          // 0.3
	LastSync              string  `json:"last_sync"`            // "2022-03-23 13:03:39"
	PowerSharingStatus    int     `json:"power_sharing_status"` // 256
	MidStatus             int     `json:"mid_status"`           // 0
	StatusID              int     `json:"status_id"`            // 194
	Name                  string  `json:"name"`
	ChargingPower         float64 `json:"charging_power"`      // 0
	MaxAvailablePower     float64 `json:"max_available_power"` // 32
	DepotName             string  `json:"depot_name"`
	ChargingSpeed         int     `json:"charging_speed"`          // 0
	AddedRange            int     `json:"added_range"`             // 91
	AddedEnergy           float64 `json:"added_energy"`            // 10.992
	AddedGreenEnergy      float64 `json:"added_green_energy"`      // 0
	AddedDischargedEnergy float64 `json:"added_discharged_energy"` // 0
	ChargingTime          int     `json:"charging_time"`           // -1648030457
	Finished              bool    `json:"finished"`                // true
	Cost                  float64 `json:"cost"`                    // 0
	CurrentMode           int     `json:"current_mode"`            // 3 (laden)
	PreventiveDischarge   bool    `json:"preventive_discharge"`    // false
	StateOfCharge         int     `json:"state_of_charge"`         // null
	OcppStatus            int     `json:"ocpp_status"`             // 1
	ConfigData            struct {
		ChargerID          int     `json:"charger_id"`
		Uid                string  `json:"uid"`
		SerialNumber       string  `json:"serial_number"`
		Name               string  `json:"name"`
		Locked             int     `json:"locked"`               // 1
		AutoLock           int     `json:"auto_lock"`            // 1
		AutoLockTime       int     `json:"auto_lock_time"`       // 60
		Multiuser          int     `json:"multiuser"`            // 1
		MaxChargingCurrent int     `json:"max_charging_current"` // 6
		Language           string  `json:"language"`             // "EN"
		IcpMaxCurrent      int     `json:"icp_max_current"`      // 0
		GridType           int     `json:"grid_type"`            // 1
		EnergyPrice        float64 `json:"energy_price"`         // 0.3
		EnergyCost         struct {
			Value            float64 `json:"value"` // 0.3
			InheritedGroupId int     `json:"inheritedGroupId"`
		} `json:"energyCost"`
		UnlockUserID        int     `json:"unlock_user_id"`
		PowerSharingConfig  int     `json:"power_sharing_config"`  // 256
		PurchasedPower      float64 `json:"purchased_power"`       // 0
		ShowName            int     `json:"show_name"`             // 1
		ShowLastname        int     `json:"show_lastname"`         // 1
		ShowEmail           int     `json:"show_email"`            // 1
		ShowProfile         int     `json:"show_profile"`          // 1
		ShowDefaultUser     int     `json:"show_default_user"`     // 1
		GestureStatus       int     `json:"gesture_status"`        // 7
		HomeSharing         int     `json:"home_sharing"`          // 0
		DcaStatus           int     `json:"dca_status"`            // 0
		ConnectionType      int     `json:"connection_type"`       // 1
		MaxAvailableCurrent int     `json:"max_available_current"` // 32
		LiveRefreshTime     int     `json:"live_refresh_time"`     // 30
		UpdateRefreshTime   int     `json:"update_refresh_time"`   // 300
		OwnerID             int     `json:"owner_id"`
		RemoteAction        int     `json:"remote_action"`        // 0
		RfidType            int     `json:"rfid_type"`            // null
		ChargerHasImage     int     `json:"charger_has_image"`    // 0
		Sha256ChargerImage  string  `json:"sha256_charger_image"` // null
		Plan                struct {
			PlanName string      `json:"plan_name"` // "Basic"
			Features interface{} `json:"features"`  // ["DEFAULT_FEATURE", "POWER_BOOST", "MOBILE_CONNECTIVITY", "AUTOMATIC_REPORTING", "STATISTICS"
		} `json:"plan"`
		SyncTimestamp int `json:"sync_timestamp"` // 1648040092
		Currency      struct {
			Id     int    `json:"id"`     // 1
			Name   string `json:"name"`   // "Euro Member Countries"
			Symbol string `json:"symbol"` // "\u20ac"
			Code   string `json:"code"`   // "EUR
		} `json:"currency"`
		ChargerLoadType           string `json:"charger_load_type"`           // "Private"
		ContractChargingAvailable bool   `json:"contract_charging_available"` // false
		Country                   struct {
			Id         int    `json:"id"`         // 4
			Code       string `json:"code"`       // "DEU"
			Iso2       string `json:"iso2"`       // "DE"
			Name       string `json:"name"`       // "ALEMANIA"
			Phone_code string `json:"phone_code"` // "49
		} `json:"country"`
		State      int    `json:"state"`       // null
		Timezone   int    `json:"timezone"`    // null
		PartNumber string `json:"part_number"` // "PLP1-0-2-4-3-SE1-A"
		Software   struct {
			UpdateAvailable bool   `json:"updateAvailable"` // false
			CurrentVersion  string `json:"currentVersion"`  // "5.5.10"
			LatestVersion   string `json:"latestVersion"`   // "5.5.10
		} `json:"software"`
		Available            int         `json:"available"`              // 1
		OperationMode        string      `json:"operation_mode"`         // "wallbox"
		OcppReady            string      `json:"ocpp_ready"`             // "ocpp_1.6j"
		Tariffs              interface{} `json:"tariffs"`                // []
		MidEnabled           int         `json:"mid_enabled"`            // 0
		MidMargin            int         `json:"mid_margin"`             // 1
		MidMarginUnit        int         `json:"mid_margin_unit"`        // 1
		MidSerialNumber      string      `json:"mid_serial_number"`      // ""
		MidStatus            int         `json:"mid_status"`             // 0
		SessionSegmentLength int         `json:"session_segment_length"` // 0
		GroupID              int         `json:"group_id"`               // 34747
	} `json:"config_data"`
}
