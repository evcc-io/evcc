package polestar

import (
	"strconv"
	"time"
)

// Timestamp is the Unix epoch timestamp used across telemetry responses
type Timestamp struct {
	Seconds string `json:"seconds"`
	Nanos   int64  `json:"nanos"`
}

// Time converts the epoch timestamp to time.Time, returning the zero value when unset
func (t Timestamp) Time() time.Time {
	sec, err := strconv.ParseInt(t.Seconds, 10, 64)
	if err != nil || sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, t.Nanos)
}

// Battery is the /telemetry/battery response payload
type Battery struct {
	BatteryChargeLevelPercentage       float64   `json:"batteryChargeLevelPercentage"`
	ChargerConnectionStatus            string    `json:"chargerConnectionStatus"`
	ChargingStatusV2                   string    `json:"chargingStatusV2"`
	EstimatedDistanceToEmptyKm         int64     `json:"estimatedDistanceToEmptyKm"`
	EstimatedChargingTimeToFullMinutes int64     `json:"estimatedChargingTimeToFullMinutes"`
	Timestamp                          Timestamp `json:"timestamp"`
}

// Odometer is the /telemetry/odometer response payload
type Odometer struct {
	OdometerMeters float64 `json:"odometerMeters"`
}

// TargetSoc is the /charging/target-soc response payload
type TargetSoc struct {
	TargetSoc struct {
		BatteryChargeTargetLevel int64 `json:"batteryChargeTargetLevel"`
	} `json:"targetSoc"`
}
