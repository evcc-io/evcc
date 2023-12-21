package hello

import (
	"strconv"
	"strings"
)

const ResponseOK = 1000

type ResponseCode int

func (rc *ResponseCode) UnmarshalJSON(data []byte) error {
	i, err := strconv.Atoi(strings.Trim(string(data), `"`))
	if err == nil {
		*rc = ResponseCode(i)
	}
	return err
}

type Error struct {
	Code    ResponseCode
	Message string
}

type AppToken struct {
	ExpiresIn    int
	AccessToken  string
	UserId       string
	RefreshToken string
}

type Vehicle struct {
	VIN string
}

type VehicleStatus struct {
	BasicVehicleStatus struct {
		UsageMode    int    `json:"usageMode,string"` // "0",
		EngineStatus string `json:"engineStatus"`     // "engine_off",
		Position     struct {
			Altitude               int  `json:"altitude,string"`               // "117",
			PosCanBeTrusted        bool `json:"posCanBeTrusted,string"`        // "true",
			Latitude               int  `json:"latitude,string"`               // "18...",
			CarLocatorStatUploadEn bool `json:"carLocatorStatUploadEn,string"` // "true",
			Longitude              int  `json:"longitude,string"`              // "28..."
		}
		DistanceToEmpty int     `json:"distanceToEmpty,string"` // "0",
		CarMode         int     `json:"carMode,string"`         // "0",
		Speed           float64 `json:"speed,string"`           // "0.0",
		SpeedValidity   bool    `json:"speedValidity,string"`   // "true",
		Direction       int     `json:"direction,string"`       // "277"
	}
	UpdateTime              int64 `json:"updateTime,string"` // "1703072512182",
	AdditionalVehicleStatus struct {
		MaintenanceStatus struct {
			TyreTempWarningPassengerRear int     `json:"tyreTempWarningPassengerRear,string"` // "0",
			DaysToService                int     `json:"daysToService,string"`                // "455",
			EngineHrsToService           int     `json:"engineHrsToService,string"`           // "500",
			Odometer                     float64 `json:"odometer,string"`                     // "7854.000",
			BrakeFluidLevelStatus        int     `json:"brakeFluidLevelStatus,string"`        // "3",
			MainBatteryStatus            struct {
				StateOfCharge int     `json:"stateOfCharge,string"` // "1",
				ChargeLevel   float64 `json:"chargeLevel,string"`   // "0.0",
				EnergyLevel   int     `json:"energyLevel,string"`   // "0",
				StateOfHealth int     `json:"stateOfHealth,string"` // "0",
				PowerLevel    int     `json:"powerLevel,string"`    // "0",
				Voltage       float64 `json:"voltage,string"`       // "5.000"
			}
		}
		ElectricVehicleStatus struct {
			DisChargeUAct                  float64 `json:"disChargeUAct,string"`                  // "0.0",
			DisChargeSts                   int     `json:"disChargeSts,string"`                   // "0",
			WptFineAlignt                  int     `json:"wptFineAlignt,string"`                  // "0",
			ChargeLidAcStatus              int     `json:"chargeLidAcStatus,string"`              // "2",
			DistanceToEmptyOnBatteryOnly   int     `json:"distanceToEmptyOnBatteryOnly,string"`   // "233",
			DistanceToEmptyOnBattery100Soc int     `json:"distanceToEmptyOnBattery100Soc,string"` // "330",
			ChargeSts                      int     `json:"chargeSts,string"`                      // "0",
			AverPowerConsumption           float64 `json:"averPowerConsumption,string"`           // "-85.5",
			ChargerState                   int     `json:"chargerState,string"`                   // "0",
			TimeToTargetDisCharged         int     `json:"timeToTargetDisCharged,string"`         // "2047",
			DistanceToEmptyOnBattery20Soc  int     `json:"distanceToEmptyOnBattery20Soc,string"`  // "66",
			DisChargeConnectStatus         int     `json:"disChargeConnectStatus,string"`         // "0",
			ChargeLidDcAcStatus            int     `json:"chargeLidDcAcStatus,string"`            // "2",
			DcChargeSts                    int     `json:"dcChargeSts,string"`                    // "0",
			PtReady                        int     `json:"ptReady,string"`                        // "0",
			ChargeLevel                    int     `json:"chargeLevel,string"`                    // "76",
			StatusOfChargerConnection      int     `json:"statusOfChargerConnection,string"`      // "0",
			DcDcActvd                      int     `json:"dcDcActvd,string"`                      // "0",
			IndPowerConsumption            int     `json:"indPowerConsumption,string"`            // "1000",
			DcDcConnectStatus              int     `json:"dcDcConnectStatus,string"`              // "0",
			DisChargeIAct                  float64 `json:"disChargeIAct,string"`                  // "0.0",
			DcChargeIAct                   float64 `json:"dcChargeIAct,string"`                   // "0.0",
			ChargeUAct                     float64 `json:"chargeUAct,string"`                     // "0.0",
			BookChargeSts                  int     `json:"bookChargeSts,string"`                  // "0",
			ChargeIAct                     float64 `json:"chargeIAct,string"`                     // "0.000",
			TimeToFullyCharged             int     `json:"timeToFullyCharged,string"`             // "2047"
		}
		ChargeHvSts   int `json:"chargeHvSts,string"` // "1",
		ClimateStatus struct {
			PreClimateActive bool `json:"preClimateActive"` // false,
			Defrost          bool `json:"defrost,string"`   // "false",
		}
	}
}
