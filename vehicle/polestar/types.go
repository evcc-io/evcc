package polestar

import "time"

type ConsumerCar struct {
	VIN                       string
	InternalVehicleIdentifier string
}

type BatteryData struct {
	BatteryChargeLevelPercentage       float64
	ChargerConnectionStatus            string
	ChargingStatus                     string
	EstimatedChargingTimeToFullMinutes int
	EstimatedDistanceToEmptyKm         int
	EventUpdatedTimestamp              EventUpdatedTimestamp
}

type OdometerData struct {
	OdometerMeters        float64
	EventUpdatedTimestamp EventUpdatedTimestamp
}

type CarTelemetryData struct {
	Battery  BatteryData
	Odometer OdometerData
}

type EventUpdatedTimestamp struct {
	ISO time.Time
	// Unix int64 `json:",string"`
}
