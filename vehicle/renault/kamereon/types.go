package kamereon

import (
	"errors"
	"strings"
)

type Response struct {
	Accounts     []Account // /commerce/v1/persons/%s
	AccessToken  string    // /commerce/v1/accounts/%s/kamereon/token
	VehicleLinks []Vehicle // /commerce/v1/accounts/%s/vehicles
	Data         Data      // /commerce/v1/accounts/%s/kamereon/kca/car-adapter/vX/cars/%s/...
}

type Account struct {
	AccountID     string
	AccountType   string
	AccountStatus string
	Country       string
	RelationType  string
}

type Vehicle struct {
	Brand           string
	VIN             string
	Status          string
	ConnectedDriver connectedDriver
}

type connectedDriver struct {
	Role string
}

func (v *Vehicle) Available() error {
	if strings.ToUpper(v.Status) != "ACTIVE" {
		return errors.New("vehicle is not active")
	}

	if v.ConnectedDriver.Role == "" {
		return errors.New("vehicle is not connected to driver")
	}

	return nil
}

type Data struct {
	Attributes attributes
}

type attributes struct {
	// battery-status
	Timestamp          string  `json:"timestamp"`
	ChargingStatus     float32 `json:"chargingStatus"`
	InstantaneousPower int     `json:"instantaneousPower"`
	RangeHvacOff       int     `json:"rangeHvacOff"`
	BatteryAutonomy    int     `json:"batteryAutonomy"`
	BatteryLevel       int     `json:"batteryLevel"`
	BatteryTemperature int     `json:"batteryTemperature"`
	PlugStatus         int     `json:"plugStatus"`
	LastUpdateTime     string  `json:"lastUpdateTime"`
	ChargePower        int     `json:"chargePower"`
	RemainingTime      *int    `json:"chargingRemainingTime"`
	// hvac-status
	ExternalTemperature float64 `json:"externalTemperature"`
	HvacStatus          string  `json:"hvacStatus"`
	// cockpit
	TotalMileage float64 `json:"totalMileage"`
	// position
	Latitude  float64 `json:"gpsLatitude"`
	Longitude float64 `json:"gpsLongitude"`
}
