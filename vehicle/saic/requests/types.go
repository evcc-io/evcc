package requests

type ChargeStatus struct {
	RvsChargeStatus struct {
		MileageSinceLastCharge    int
		TotalBatteryCapacity      int
		WorkingVoltage            int
		ChargingDuration          int
		ChargingType              int
		LastChargeEndingPower     int
		FuelRangeElec             int64 // Value / 10 = Range
		RealtimePower             int
		WorkingCurrent            int
		ChargingGunState          int // Gun connected
		MileageOfDay              int
		StartTime                 int64
		EndTime                   int64
		PowerUsageOfDay           int
		PowerUsageSinceLastCharge int
		Mileage                   uint // Odometer
	}
	ChrgMgmtData struct {
		BmsChrgOtptCrntReqV       int
		CcuOnbdChrgrPlugOn        int
		BmsPTCHeatResp            int
		BmsChrgSts                int
		ChrgngAddedElecRngV       int
		BmsPackSOCDsp             int // SOC in per mille
		CcuOffBdChrgrPlugOn       int
		BmsPackCrnt               int
		ImcuChrgngEstdElecRng     int
		ChrgngSpdngTimeV          int
		BmsReserStMintueDspCmd    int
		BmsDsChrgSpRsn            int
		ImcuChrgngEstdElecRngV    int
		BmsPackVol                int
		BmsPackCrntV              int
		ImcuVehElecRngV           int
		BmsReserSpHourDspCmd      int
		BmsAdpPubChrgSttnDspCmd   int
		BmsReserSpMintueDspCmd    int
		BmsAltngChrgCrntDspCmd    int
		DisChrgngRmnngTimeV       int
		ImcuDschrgngEstdElecRngV  int
		BmsChrgSpRsn              int
		DisChrgngRmnngTime        int
		ChrgngAddedElecRng        int
		ClstrElecRngToEPT         int // Range
		BmsPTCHeatReqDspCmd       int
		CcuEleccLckCtrlDspCmd     int
		BmsChrgCtrlDspCmd         int
		BmsOnBdChrgTrgtSOCDspCmd  int
		OnBdChrgrAltrCrntInptCrnt int
		BmsEstdElecRng            int
		ChrgngDoorOpenCnd         int
		BmsReserCtrlDspCmd        int
		BmsChrgOtptCrntReq        int
		ChrgngDoorPosSts          int
		BmsReserStHourDspCmd      int
		ImcuVehElecRng            int // Range
		ChrgngSpdngTime           int
		ChrgngRmnngTimeV          int
		ImcuDschrgngEstdElecRng   int
		ChrgngRmnngTime           int64 // Charging remaining time
		OnBdChrgrAltrCrntInptVol  int
	}
}

type LoginData struct {
	Tenant_id     string
	LanguageType  string
	User_name     string
	Avatar        string
	Token_type    string `json:"token_type,omitempty"`
	Client_id     string
	Access_token  string `json:"access_token"`
	Role_name     string
	Refresh_token string `json:"refresh_token,omitempty"`
	License       string
	Post_id       string
	User_id       string
	Role_id       string
	Scope         string
	Oauth_id      string
	Detail        struct {
		LanguageType string
	}
	Dept_id    string
	Expires_in int64 `json:"expires_in,omitempty"`
	Aaccount   string
	Jti        string
}

type Answer struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}
