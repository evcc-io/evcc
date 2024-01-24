package psa

import "time"

// Vehicle is a single vehicle
type Vehicle struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Pictures []string `json:"pictures"`
	VIN      string   `json:"vin"`
}

// Status is the /status response
type Status struct {
	Battery struct {
		Capacity int64
		Health   struct {
			Capacity   int64
			Resistance int64
		}
	}
	Charging struct {
		ChargingMode    string
		ChargingRate    int64
		NextDelayedTime string
		Plugged         bool
		RemainingTime   string
		Status          string
	}
	Preconditionning struct {
		AirConditioning struct {
			UpdatedAt time.Time
			Status    string // Disabled
		}
	}
	Energy   []Energy
	Odometer struct {
		Mileage float64
	}
	LastPosition struct {
		Type     string
		Geometry struct {
			Type        string
			Coordinates []float64
		}
		Properties struct {
			UpdatedAt time.Time
			Type      string
			Heading   int
		}
	}
}

// Energy is the /status partial energy response
type Energy struct {
	UpdatedAt time.Time
	Type      string // Fuel/Electric
	Level     float64
	Autonomy  int
	Charging  struct {
		Plugged         bool
		Status          string // InProgress
		RemainingTime   Duration
		ChargingRate    int
		ChargingMode    string // "Slow"
		NextDelayedTime Duration
	}
}
