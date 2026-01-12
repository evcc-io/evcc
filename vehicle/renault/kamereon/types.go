package kamereon

import (
	"errors"
	"strings"
)

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

	// DEPRECATED
	// if v.ConnectedDriver.Role == "" {
	// 	return errors.New("vehicle is not connected to driver")
	// }

	return nil
}

type BatteryStatus struct {
	Timestamp          string  `json:"timestamp"`
	ChargingStatus     float32 `json:"chargingStatus"`
	InstantaneousPower int     `json:"instantaneousPower"`
	RangeHvacOff       int     `json:"rangeHvacOff"`
	BatteryAutonomy    int     `json:"batteryAutonomy"`
	BatteryLevel       *int    `json:"batteryLevel"`
	BatteryTemperature int     `json:"batteryTemperature"`
	PlugStatus         int     `json:"plugStatus"`
	LastUpdateTime     string  `json:"lastUpdateTime"`
	ChargePower        int     `json:"chargePower"`
	RemainingTime      *int    `json:"chargingRemainingTime"`
}

type HvacStatus struct {
	ExternalTemperature float64 `json:"externalTemperature"`
	HvacStatus          string  `json:"hvacStatus"`
}

type Cockpit struct {
	TotalMileage *float64 `json:"totalMileage"`
}

type SocLevels struct {
	SocMin                    *int   `json:"socMin"`
	SocTarget                 *int   `json:"socTarget"`
	LastEnergyUpdateTimestamp string `json:"lastEnergyUpdateTimestamp"`
}

type Position struct {
	Latitude  float64 `json:"gpsLatitude"`
	Longitude float64 `json:"gpsLongitude"`
}

type ChargeAction struct {
	Type       string                 `json:"type"`
	Attributes ChargeActionAttributes `json:"attributes"`
}

type ChargeActionAttributes struct {
	Action string `json:"action"`
}

type DataEnvelope[T any] struct {
	Data struct {
		Attributes T `json:"attributes"`
	} `json:"data"`
}

type EvSettingsRequest struct {
	LastSettingsUpdateTimestamp    string  `json:"lastSettingsUpdateTimestamp"`
	DelegatedActivated             bool    `json:"delegatedActivated"`
	ChargeModeRq                   string  `json:"chargeModeRq"`
	ChargeTimeStart                string  `json:"chargeTimeStart"`
	ChargeDuration                 int     `json:"chargeDuration"`
	PreconditioningTemperature     float64 `json:"preconditioningTemperature"`
	PreconditioningHeatedStrgWheel bool    `json:"preconditioningHeatedStrgWheel"`
	PreconditioningHeatedRightSeat bool    `json:"preconditioningHeatedRightSeat"`
	PreconditioningHeatedLeftSeat  bool    `json:"preconditioningHeatedLeftSeat"`
	Programs                       []any   `json:"programs"`
}

type EvSettingsResponse struct {
	CommandId string `json:"commandId"`
	Type      string `json:"type"`
	Status    string `json:"status"`
}
