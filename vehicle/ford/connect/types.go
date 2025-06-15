package connect

import (
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
	Timestamp       time.Time `json:",format:'01-02-2006 15:04:05'"`
}

type VehicleStatus struct {
	ChargingStatus struct {
		Value           string    // "NotReady",
		TimeStamp       time.Time `json:",format:'01-02-2006 15:04:05'"`
		ChargeStartTime time.Time `json:",format:'01-02-2006 15:04:05'"`
		ChargeEndTime   time.Time `json:",format:'01-02-2006 15:04:05'"`
	}
	PlugStatus struct {
		Value     bool      // false,
		TimeStamp time.Time `json:",format:'01-02-2006 15:04:05'"`
	}
}

type VehicleLocation struct {
	Speed     float64   // 0,
	Direction string    // "SOUTHEAST",
	TimeStamp time.Time `json:",format:'01-02-2006 15:04:05'"`
	Longitude float64   `json:",string"`
	Latitude  float64   `json:",string"`
}
