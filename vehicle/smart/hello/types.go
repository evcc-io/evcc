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

type Int int

func (rc *Int) UnmarshalJSON(data []byte) error {
	i, err := strconv.Atoi(strings.Trim(string(data), `"`))
	if err != nil && string(data) != "" {
		return err
	}
	*rc = Int(i)
	return nil
}

type VehicleStatus struct {
	BasicVehicleStatus struct {
		UsageMode    Int    `json:"usageMode"`    // "0",
		EngineStatus string `json:"engineStatus"` // "engine_off",
		Position     struct {
			Altitude               Int  `json:"altitude"`                      // "117",
			PosCanBeTrusted        bool `json:"posCanBeTrusted,string"`        // "true",
			Latitude               Int  `json:"latitude"`                      // "18...",
			CarLocatorStatUploadEn bool `json:"carLocatorStatUploadEn,string"` // "true",
			Longitude              Int  `json:"longitude"`                     // "28..."
		}
		DistanceToEmpty Int     `json:"distanceToEmpty"`      // "0",
		CarMode         Int     `json:"carMode"`              // "0",
		Speed           float64 `json:"speed,string"`         // "0.0",
		SpeedValidity   bool    `json:"speedValidity,string"` // "true",
		Direction       Int     `json:"direction"`            // "277"
	}
	UpdateTime              int64 `json:"updateTime,string"` // "1703072512182",
	AdditionalVehicleStatus struct {
		MaintenanceStatus struct {
			TyreTempWarningPassengerRear Int     `json:"tyreTempWarningPassengerRear"` // "0",
			DaysToService                Int     `json:"daysToService"`                // "455",
			EngineHrsToService           Int     `json:"engineHrsToService"`           // "500",
			Odometer                     float64 `json:"odometer,string"`              // "7854.000",
			BrakeFluidLevelStatus        Int     `json:"brakeFluidLevelStatus"`        // "3",
			MainBatteryStatus            struct {
				StateOfCharge Int     `json:"stateOfCharge"`      // "1",
				ChargeLevel   float64 `json:"chargeLevel,string"` // "0.0",
				EnergyLevel   Int     `json:"energyLevel"`        // "0",
				StateOfHealth Int     `json:"stateOfHealth"`      // "0",
				PowerLevel    Int     `json:"powerLevel"`         // "0",
				Voltage       float64 `json:"voltage,string"`     // "5.000"
			}
		}
		ElectricVehicleStatus struct {
			DisChargeUAct                  float64 `json:"disChargeUAct,string"`           // "0.0",
			DisChargeSts                   Int     `json:"disChargeSts"`                   // "0",
			WptFineAlignt                  Int     `json:"wptFineAlignt"`                  // "0",
			ChargeLidAcStatus              Int     `json:"chargeLidAcStatus"`              // "2",
			DistanceToEmptyOnBatteryOnly   Int     `json:"distanceToEmptyOnBatteryOnly"`   // "233",
			DistanceToEmptyOnBattery100Soc Int     `json:"distanceToEmptyOnBattery100Soc"` // "330",
			ChargeSts                      Int     `json:"chargeSts"`                      // "0",
			AverPowerConsumption           float64 `json:"averPowerConsumption,string"`    // "-85.5",
			ChargerState                   Int     `json:"chargerState"`                   // "0",
			TimeToTargetDisCharged         Int     `json:"timeToTargetDisCharged"`         // "2047",
			DistanceToEmptyOnBattery20Soc  Int     `json:"distanceToEmptyOnBattery20Soc"`  // "66",
			DisChargeConnectStatus         Int     `json:"disChargeConnectStatus"`         // "0",
			ChargeLidDcAcStatus            Int     `json:"chargeLidDcAcStatus"`            // "2",
			DcChargeSts                    Int     `json:"dcChargeSts"`                    // "0",
			PtReady                        Int     `json:"ptReady"`                        // "0",
			ChargeLevel                    Int     `json:"chargeLevel"`                    // "76",
			StatusOfChargerConnection      Int     `json:"statusOfChargerConnection"`      // "0",
			DcDcActvd                      Int     `json:"dcDcActvd"`                      // "0",
			IndPowerConsumption            Int     `json:"indPowerConsumption"`            // "1000",
			DcDcConnectStatus              Int     `json:"dcDcConnectStatus"`              // "0",
			DisChargeIAct                  float64 `json:"disChargeIAct,string"`           // "0.0",
			DcChargeIAct                   float64 `json:"dcChargeIAct,string"`            // "0.0",
			ChargeUAct                     float64 `json:"chargeUAct,string"`              // "0.0",
			BookChargeSts                  Int     `json:"bookChargeSts"`                  // "0",
			ChargeIAct                     float64 `json:"chargeIAct,string"`              // "0.000",
			TimeToFullyCharged             Int     `json:"timeToFullyCharged"`             // "2047"
		}
		ChargeHvSts   Int `json:"chargeHvSts"` // "1",
		ClimateStatus struct {
			PreClimateActive bool `json:"preClimateActive"` // false,
			Defrost          bool `json:"defrost,string"`   // "false",
		}
	}
}
