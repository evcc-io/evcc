package tronity

// https://app.platform.tronity.io/docs#operation

const (
	ReadBattery          = "read_battery"            // Read EV battery's capacity and state of charge
	ReadCharge           = "read_charge"             // Know whether vehicle is charging
	ReadLocation         = "read_location"           // Access location
	ReadOdometer         = "read_odometer"           // Retrieve total distance traveled
	ReadVehicleInfo      = "read_vehicle_info"       // Know make, model, and year
	ReadVIN              = "read_vin"                // Read VIN
	WriteChargeStartStop = "write_charge_start_stop" // Start or stop your vehicle's charging
	WriteWakeUp          = "write_wake_up"           // Wake up car. Only valid for Tesla
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
	VIN      string
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
