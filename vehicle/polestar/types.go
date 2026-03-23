package polestar

type ConsumerCar struct {
	VIN                       string
	InternalVehicleIdentifier string
}

type Timestamp struct {
	Seconds string
	Nanos   int64
}

type HealthData struct {
	VIN                       string
	BrakeFluidLevelWarning    string
	DaysToService             int64
	DistanceToServiceKm       int64
	EngineCoolantLevelWarning string
	OilLevelWarning           string
	ServiceWarning            string
	Timestamp                 Timestamp
}

type BatteryData struct {
	VIN                                string
	BatteryChargeLevelPercentage       float64
	ChargingStatus                     string
	EstimatedChargingTimeToFullMinutes int64
	EstimatedDistanceToEmptyKm         int64
	Timestamp                          Timestamp
}

type OdometerData struct {
	VIN            string
	OdometerMeters int64
	Timestamp      Timestamp
}

type CarTelemetryData struct {
	Health   []HealthData
	Battery  []BatteryData
	Odometer []OdometerData
}
