package mercedes

import (
	"fmt"
	"html"
	"time"

	"golang.org/x/oauth2"
)

var Regions = map[string]string{
	"apac":  "Asia-Pacific",
	"ece":   "ECE",
	"noram": "North-America",
}

type MBToken struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	ExpiresIn int    `json:"expires_in"`
	oauth2.Token
}

func (t *MBToken) GetToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.Access,
		RefreshToken: t.Refresh,
		Expiry:       time.Now().Add(time.Duration(t.ExpiresIn) * time.Second),
	}
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

type PinResponse struct {
	IsEmail  bool   `json:"isEmail"`
	UserName string `json:"username"`
}

type VehiclesResponse struct {
	assignedVehicles []Vehicle
}

type Vehicle struct {
	fin string
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
