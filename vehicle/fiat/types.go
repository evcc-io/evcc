package fiat

const (
	LoginURI = "https://loginmyuconnect.fiat.com"
	ApiURI   = "https://channels.sdpr-01.fcagcv.com"
	ApiKey   = "3_mOx_J2dRgjXYCdyhchv3b5lhi54eBcdCTX4BI8MORqmZCoQWhA0mV2PTlptLGUQI"
)

type Vehicle struct {
	VIN string
}

type Vehicles struct {
	Vehicles []Vehicle
}

type Status struct {
	EvInfo struct {
		Battery struct {
			ChargingLevel   string // LEVEL_2
			ChargingStatus  string // CHARGING
			DistanceToEmpty struct {
				Value int
				Unit  string
			}
			PlugInStatus        bool // true
			StateOfCharge       int  // 75
			TimeToFullyChargeL1 int  // 0
			TimeToFullyChargeL2 int  // 540
			TotalRange          int  // 17
		}
		Timestamp int64
	}
}
