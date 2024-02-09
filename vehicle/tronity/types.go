package tronity

// https://app.tronity.tech/docs#section/Authentication-Flow

const (
	ReadCharge           = "tronity_charging" // Know whether vehicle is charging
	ReadOdometer         = "tronity_odometer" // Retrieve total distance traveled
	WriteChargeStartStop = "tronity_control_charging"
)

type Vehicles struct {
	Data []Vehicle
}

type Vehicle struct {
	ID          string
	VIN         string
	DisplayName string
	Manufacture string
	Scopes      []string
}

type Bulk struct {
	Odometer float64
	Range    float64
	Level    float64
	Charging string // Charging
	// Latitude  float64/string
	// Longitude float64/string
	Timestamp int64
}

type Odometer struct {
	Odometer  float64
	Timestamp float64
}

type EVBatteryLevel struct {
	Range     float64
	Level     float64
	Timestamp int64
}

type EVChargingStatus struct {
	Charging  string
	Timestamp int64
}

type Location struct {
	// Latitude  float64/string
	// Longitude float64/string
	Timestamp int64
}
