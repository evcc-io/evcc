package polestar

import "time"

type ConsumerCar struct {
	VIN                       string
	InternalVehicleIdentifier string
}

type BatteryData struct {
	BatteryChargeLevelPercentage       int
	ChargerConnectionStatus            string
	ChargingStatus                     string
	EstimatedChargingTimeToFullMinutes int
	EstimatedDistanceToEmptyKm         int
	EventUpdatedTimestamp              EventUpdatedTimestamp
}

type OdometerData struct {
	OdometerMeters        int
	EventUpdatedTimestamp EventUpdatedTimestamp
}

type EventUpdatedTimestamp struct {
	ISO  time.Time
	Unix uint64
}
