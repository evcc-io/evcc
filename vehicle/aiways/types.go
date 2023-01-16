package aiways

type User struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		Email        string      `json:"email"`
		HeadURL      string      `json:"headUrl"`
		IsFirstLogin int64       `json:"isFirstLogin"`
		Mobile       interface{} `json:"mobile"`
		Nickname     string      `json:"nickname"`
		Sex          interface{} `json:"sex"`
		Token        string      `json:"token"`
		UserID       int64       `json:"userId"`
		UserType     int64       `json:"userType"`
		Username     string      `json:"username"`
	} `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

type StatusResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		Vc struct {
			DataTime   string `json:"dataTime"`
			DataTimeTS int64  `json:"dataTimeTS"`
			// NorthLat               string `json:"northLat"`
			// EastLon                string `json:"eastLon"`
			Lat       float64 `json:"lat,string"`
			Lon       float64 `json:"lon,string"`
			Address   string  `json:"address"`
			ChargeSts int     `json:"chargeSts,string"`
			// SocLow                 string  `json:"socLow"`
			// LastFlameoutTime       string  `json:"lastFlameoutTime"`
			// VehicleSts             string  `json:"vehicleSts"`
			// LowSpdMuteSts          string  `json:"lowSpdMuteSts"`
			// DoorLockSts            string  `json:"doorLockSts"`
			// TrunkLockSts           string  `json:"trunkLockSts"`
			// DoorStsReserved        string  `json:"doorStsReserved"`
			// DoorOpenStsLf          string  `json:"doorOpenStsLf"`
			// DoorOpenStsRf          string  `json:"doorOpenStsRf"`
			// DoorOpenStsLb          string  `json:"doorOpenStsLb"`
			// DoorOpenStsRb          string  `json:"doorOpenStsRb"`
			// FrontHatchCoverOpenSts string  `json:"frontHatchCoverOpenSts"`
			// TrunkDoorOpenSts       string  `json:"trunkDoorOpenSts"`
			// ChargingCoverOpenSts   string  `json:"chargingCoverOpenSts"`
			// WindowOpenStsLf        string  `json:"windowOpenStsLf"`
			// WindowOpenStsRf        string  `json:"windowOpenStsRf"`
			// WindowOpenStsLb        string  `json:"windowOpenStsLb"`
			// WindowOpenStsRb        string  `json:"windowOpenStsRb"`
			// LeftTurnLightOpenSts   string  `json:"leftTurnLightOpenSts"`
			// RightTurnLightOpenSts  string  `json:"rightTurnLightOpenSts"`
			// BackFrogLightOpenSts   string  `json:"backFrogLightOpenSts"`
			// HighLightOpenSts       string  `json:"highLightOpenSts"`
			// LowLightOpenSts        string  `json:"lowLightOpenSts"`
			// WidthLightOpenSts      string  `json:"widthLightOpenSts"`
			// BrakeLightOpenSts      string  `json:"brakeLightOpenSts"`
			// ReversingLightOpenSts  string  `json:"reversingLightOpenSts"`
			// DayRunningLightOpenSts string  `json:"dayRunningLightOpenSts"`
			// AutoLightOpenSts       string  `json:"autoLightOpenSts"`
			// SunroofStsSts          string  `json:"sunroofStsSts"`
			// SunroofHorizontalOpen  string  `json:"sunroofHorizontalOpen"`
			// SunroofSunshadeOpen    string  `json:"sunroofSunshadeOpen"`
			// TirePressureStsLf      string  `json:"tirePressureStsLf"`
			// TirePressureStsRf      string  `json:"tirePressureStsRf"`
			// TirePressureStsLb      string  `json:"tirePressureStsLb"`
			// TirePressureStsRb      string  `json:"tirePressureStsRb"`
			// TirePressureLf         string  `json:"tirePressureLf"`
			// TirePressureRf         string  `json:"tirePressureRf"`
			// TirePressureLb         string  `json:"tirePressureLb"`
			// TirePressureRb         string  `json:"tirePressureRb"`
			// TireTempHighStsLf      string  `json:"tireTempHighStsLf"`
			// TireTempHighStsRf      string  `json:"tireTempHighStsRf"`
			// TireTempHighStsLb      string  `json:"tireTempHighStsLb"`
			// TireTempHighStsRb      string  `json:"tireTempHighStsRb"`
			// TireTempLf             string  `json:"tireTempLf"`
			// TireTempRf             string  `json:"tireTempRf"`
			// TireTempLb             string  `json:"tireTempLb"`
			// TireTempRb             string  `json:"tireTempRb"`
			// AirconAcSts            string  `json:"airconAcSts"`
			// AirconAutoSts          string  `json:"airconAutoSts"`
			// AirconWindMode         string  `json:"airconWindMode"`
			// AirconWindVolume       string  `json:"airconWindVolume"`
			// AirconCycleSts         string  `json:"airconCycleSts"`
			// AirconTempDisLf        string  `json:"airconTempDisLf"`
			// AirconTempDisRf        string  `json:"airconTempDisRf"`
			// AirconSyncSts          string  `json:"airconSyncSts"`
			// AirconRunSts           string  `json:"airconRunSts"`
			// AirconOutsideTemp      string  `json:"airconOutsideTemp"`
			// AirconInsideTemp       string  `json:"airconInsideTemp"`
			// SafetyBeltStsDrv       string  `json:"safetyBeltStsDrv"`
			// SafetyBeltStsPass      string  `json:"safetyBeltStsPass"`
			// SeatBeltStsBl          string  `json:"seatBeltStsBl"`
			// SeatBeltStsBm          string  `json:"seatBeltStsBm"`
			// SeatBeltStsBr          string  `json:"seatBeltStsBr"`
			// SeatHeatStsDrv         string  `json:"seatHeatStsDrv"`
			// SeatHeatStsPass        string  `json:"seatHeatStsPass"`
			// SeatHeatStsSys         string  `json:"seatHeatStsSys"`
			// ChgConnStsDc           string  `json:"chgConnStsDc"`
			// ChgConnStsAc           string  `json:"chgConnStsAc"`
			// KeyLowPowerWarn        string  `json:"keyLowPowerWarn"`
			DrivingRange   float64 `json:"drivingRange,string"`
			VehicleMileage float64 `json:"vehicleMileage,string"`
			// Speed          string `json:"speed"`
			// CrashSts       string `json:"crashSts"`
			// ElecModeFlg    string `json:"elecModeFlg"`
			// PowerMode      string `json:"powerMode"`
			// SteeringAngle  string `json:"steeringAngle"`
			// EpbSts         string `json:"epbSts"`
			Soc int `json:"soc,string"`
			// AgpsSts            string `json:"agpsSts"`
			// BtConnMac          string `json:"btConnMac"`
			// BtConnUserID       string `json:"btConnUserId"`
			// WifiConnCount      string `json:"wifiConnCount"`
			// HasSkylight        string `json:"hasSkylight"`
			// Iccid              string `json:"iccid"`
			// DataFlow           string `json:"dataFlow"`
			// AuthStatus         string `json:"authStatus"`
			// SenceSeason        int    `json:"senceSeason"`
			// SenceSeasonText    string `json:"senceSeasonText"`
			// CarSenceSeasonText string `json:"carSenceSeasonText"`
			// BmsChgRemTime      string `json:"bmsChgRemTime"`
			// BmsPreThemalMode   string `json:"bmsPreThemalMode"`
			// BcmRemoteControlSt string `json:"bcmRemoteControlSt"`
			Active bool `json:"active"`
		} `json:"vc"`
	} `json:"data"`
}
