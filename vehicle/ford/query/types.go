package query

import (
	"time"
)

type Vehicle struct {
	VIN                           string
	VehicleID                     string
	Make                          string
	ModelName                     string
	ModelCode                     string
	ModelYear                     string
	Color                         string
	NickName                      string
	VehicleAuthorizationIndicator int
	EngineType                    string
}

type FloatValue struct {
	UpdateTime time.Time
	Value      float64
}

type StringValue struct {
	UpdateTime time.Time
	Value      string
}

type Telemetry struct {
	UpdateTime time.Time
	VehicleId  string
	VIN        string
	Metrics    struct {
		DoorLockStatus any
		DoorStatus     any
		IgnitionStatus StringValue
		Position       struct {
			UpdatedTime time.Time
			Value       struct {
				Location struct {
					Lat, Lon, Alt float64
				}
			}
		}
		Odometer                      FloatValue
		XevBatteryRange               FloatValue
		XevBatteryStateOfCharge       FloatValue
		XevPlugChargerStatus          StringValue
		XevBatteryChargeDisplayStatus StringValue
		XevChargeStationPowerType     StringValue
		XevBatteryChargerEnergyOutput FloatValue
		XevBatteryTimeToFullCharge    FloatValue
	}
}
