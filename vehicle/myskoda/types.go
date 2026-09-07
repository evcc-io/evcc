package myskoda

import "time"

// VehicleResponse is the /api/v1/vehicles/{vin} response
type VehicleResponse struct {
	Vehicle Vehicle
	Errors  []Error
}

// Error describes a part of the vehicle data that could not be retrieved
type Error struct {
	Type        string
	Description string
}

// Vehicle is the vehicle and its current state
type Vehicle struct {
	VIN             string
	Name            string
	LicensePlate    string
	Odometer        *Odometer
	AirConditioning *AirConditioning
	Charging        *Charging
}

type Odometer struct {
	MileageInKm int64
}

type AirConditioning struct {
	State string
}

type Charging struct {
	Status   *ChargingStatus
	Settings *ChargingSettings
}

type ChargingStatus struct {
	ChargingRateInKilometersPerHour      float64
	ChargePowerInKw                      float64
	RemainingTimeToFullyChargedInMinutes int64
	FullyChargedAt                       time.Time
	State                                string
	ChargeType                           string
	Battery                              BatteryStatus
}

type BatteryStatus struct {
	RemainingCruisingRangeInMeters int64
	StateOfChargeInPercent         int
}

type ChargingSettings struct {
	TargetStateOfChargeInPercent *int
	MaxChargeCurrentAc           string
	MaxChargeCurrentAcAmpere     int
}
