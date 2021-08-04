package tronity

type Vehicles struct {
	Data []Vehicle
}

type Vehicle struct {
	ID          string
	VIN         string
	DisplayName string
	Manufacture string
}

type Bulk struct {
	VIN       string
	Odometer  int
	Range     int
	Level     int
	Charging  string // Charging
	Latitude  float64
	Longitude float64
	Timestamp int64
}

type Odometer struct {
	Odometer  int
	Timestamp int64
}

type EVBatteryLevel struct {
	Range     int
	Level     int
	Timestamp int64
}

type EVChargingStatus struct {
	Charging  string
	Timestamp int64
}
