package polestar

import "time"

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type ConsumerCar struct {
	VIN                       string
	InternalVehicleIdentifier string
}

type BatteryData struct {
	BatteryChargeLevelPercentage       float64
	ChargerConnectionStatus            string
	ChargingStatus                     string
	EstimatedChargingTimeToFullMinutes int
	EstimatedDistanceToEmptyKm         int
	EventUpdatedTimestamp              EventUpdatedTimestamp
}

type OdometerData struct {
	OdometerMeters        float64
	EventUpdatedTimestamp EventUpdatedTimestamp
}

type EventUpdatedTimestamp struct {
	ISO time.Time
	// Unix int64 `json:",string"`
}
