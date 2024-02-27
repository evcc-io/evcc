package mercedes

import (
	"fmt"
	"html"
	"time"
)

var Regions = map[string]string{
	"apac":  "Asia-Pacific",
	"ece":   "ECE",
	"noram": "North-America",
}

type ErrorInfo struct {
	ErrorCode    int
	ErrorMessage string
	ErrorDetails string
}

func (e ErrorInfo) Error() error {
	if e.ErrorCode == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", e.ErrorMessage, html.UnescapeString(e.ErrorDetails))
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
			StateOfCharge   float64 // 75
			EndOfChargeTime int     // Minutes after midnight
			TotalRange      int     // 17
		}
		Timestamp time.Time
	}
	LocationResponse struct {
		TimeStamp time.Time
		Longitude float64
		Latitude  float64
	}
	Timestamp time.Time
}
