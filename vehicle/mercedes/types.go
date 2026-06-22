package mercedes

import (
	"time"
)

var Regions = map[string]string{
	"apac":  "Asia-Pacific",
	"ece":   "ECE",
	"noram": "North-America",
}

type PinRequest struct {
	EmailOrPhoneNumber string `json:"emailOrPhoneNumber"`
	CountryCode        string `json:"countryCode"`
	Nonce              string `json:"nonce"`
}

type PinResponse struct {
	IsEmail  bool   `json:"isEmail"`
	UserName string `json:"username"`
}

type VehiclesResponse struct {
	AssignedVehicles []Vehicle
}

type Vehicle struct {
	Fin string
	Vin string
}

type StatusResponse struct {
	VehicleInfo struct {
		Odometer struct {
			Value int
			Unit  string
		}
		Timestamp time.Time
	}
	EvInfo struct {
		Battery struct {
			ChargingStatus  int
			DistanceToEmpty struct {
				Value int
				Unit  string
			}
			StateOfCharge         float64 // 75
			EndOfChargeTime       int     // Minutes after midnight
			TotalRange            int     // 17
			SocLimit              int     // 50-100
			SelectedChargeProgram int
		}
		Timestamp time.Time
	}
	LocationResponse struct {
		TimeStamp time.Time
		Longitude float64
		Latitude  float64
	}
	Preconditioning struct {
		Active    bool
		Timestamp time.Time
	}
	Timestamp time.Time
}
