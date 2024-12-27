package hello

import (
	"strconv"
	"strings"
)

const ResponseOK = 1000

type Int int

func (rc *Int) UnmarshalJSON(data []byte) error {
	plain := strings.Trim(string(data), `"`)
	if plain == "" {
		*rc = Int(0)
		return nil
	}
	v, err := strconv.Atoi(plain)
	if err == nil {
		*rc = Int(v)
	}
	return err
}

type Bool bool

func (rc *Bool) UnmarshalJSON(data []byte) error {
	plain := strings.Trim(string(data), `"`)
	if plain == "" {
		*rc = Bool(false)
		return nil
	}
	v, err := strconv.ParseBool(plain)
	if err == nil {
		*rc = Bool(v)
	}
	return err
}

type Error struct {
	Code    Int
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
		UsageMode    Int    `json:"usageMode"`    // "0",
		EngineStatus string `json:"engineStatus"` // "engine_off",
		Position     struct {
			Altitude               Int  `json:"altitude"`               // "117",
			PosCanBeTrusted        Bool `json:"posCanBeTrusted"`        // "true",
			Latitude               Int  `json:"latitude"`               // "18...",
			CarLocatorStatUploadEn Bool `json:"carLocatorStatUploadEn"` // "true",
			Longitude              Int  `json:"longitude"`              // "28..."
		}
		DistanceToEmpty Int     `json:"distanceToEmpty"` // "0",
		CarMode         Int     `json:"carMode"`         // "0",
		Speed           float64 `json:"speed,string"`    // "0.0",
		SpeedValidity   Bool    `json:"speedValidity"`   // "true",
		Direction       Int     `json:"direction"`       // "277"
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
			IndPowerConsumption            float64 `json:"indPowerConsumption,string"`     // "1000",
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
			PreClimateActive Bool `json:"preClimateActive"` // false,
			Defrost          Bool `json:"defrost"`          // "false",
		}
	}
}
