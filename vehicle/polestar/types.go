package polestar

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
}

type OdometerData struct {
	OdometerMeters int
}
