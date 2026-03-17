package loadpoint

import (
	"time"
)

// ThresholdsConfig defines pv mode hysteresis parameters
type ThresholdsConfig struct {
	Enable  ThresholdConfig `json:"enable"`
	Disable ThresholdConfig `json:"disable"`
}

// ThresholdConfig defines enable/disable hysteresis parameters
type ThresholdConfig struct {
	Delay     time.Duration `json:"delay"`
	Threshold float64       `json:"threshold"`
}

// SocConfig defines soc settings, estimation and update behavior
type SocConfig struct {
	Poll     PollConfig `json:"poll"`
	Estimate *bool      `json:"estimate"`
}

// GeofenceConfig defines geofence settings for vehicle detection
type GeofenceConfig struct {
	Enabled bool    `json:"enabled"`
	Lat     float64 `json:"lat"`    // Latitude of loadpoint position
	Lon     float64 `json:"lon"`    // Longitude of loadpoint position
	Radius  float64 `json:"radius"` // Maximum vehicle distance from loadpoint (m)
}

// PollConfig defines the vehicle polling mode and interval
type PollConfig struct {
	Mode     PollMode      `json:"mode"`     // polling mode charging (default), connected, always
	Interval time.Duration `json:"interval"` // interval when not charging
}

//go:generate go tool enumer -type PollMode -trimprefix Poll -transform=lower -text
type PollMode int

// Poll modes
const (
	PollCharging PollMode = iota
	PollConnected
	PollAlways
)
