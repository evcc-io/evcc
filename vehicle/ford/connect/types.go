package connect

import (
	"strings"
	"time"
)

const StatusSuccess = "SUCCESS"

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type InformationResponse struct {
	Status  string
	Vehicle Vehicle
}

type Vehicle struct {
	VehicleID                     string
	Make                          string
	ModelName                     string
	ModelYear                     string
	Color                         string
	NickName                      string
	LastUpdated                   string
	VehicleAuthorizationIndicator int
	ServiceCompatible             bool
	EngineType                    string
	VehicleDetails                VehicleDetails
	VehicleStatus                 VehicleStatus
	VehicleLocation               VehicleLocation
}
type VehicleDetails struct {
	FuelLevel, BatteryChargeLevel TimedValue
	Mileage, Odometer             float64
}

type TimedValue struct {
	Value           float64
	DistanceToEmpty float64
	Timestamp       Timestamp // "05-24-2024 15:58:56"
}

type VehicleStatus struct {
	ChargingStatus struct {
		Value           string    // "NotReady",
		TimeStamp       Timestamp // "05-24-2024 15:58:56",
		ChargeStartTime Timestamp // "01-01-2010 00:00:00",
		ChargeEndTime   Timestamp // "05-24-2024 15:33:00"
	}
	PlugStatus struct {
		Value     bool      // false,
		TimeStamp Timestamp // "05-24-2024 15:58:56"
	}
}

type VehicleLocation struct {
	Speed     float64   // 0,
	Direction string    // "SOUTHEAST",
	TimeStamp Timestamp // "05-24-2024 15:58:56",
	Longitude float64   `json:",string"`
	Latitude  float64   `json:",string"`
}
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes Ford timestamps into time.Time
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	t, err := time.Parse("01-02-2006 15:04:05", strings.Trim(string(data), `"`))

	if err == nil {
		ts.Time = t
	}

	return err
}
