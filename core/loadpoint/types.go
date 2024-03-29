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
	Delay     time.Duration
	Threshold float64
}

// SocConfig defines soc settings, estimation and update behavior
type SocConfig struct {
	Poll     PollConfig `mapstructure:"poll"`
	Estimate *bool      `mapstructure:"estimate"`
}

// PollConfig defines the vehicle polling mode and interval
type PollConfig struct {
	Mode     PollMode      `mapstructure:"mode"`     // polling mode charging (default), connected, always
	Interval time.Duration `mapstructure:"interval"` // interval when not charging
}

//go:generate enumer -type PollMode -transform=lower
type PollMode int

// Poll modes
const (
	PollCharging PollMode = iota
	PollConnected
	PollAlways
)
