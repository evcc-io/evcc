package loadpoint

import "time"

type Thresholds struct {
	Enable  ThresholdConfig `json:"enable"`
	Disable ThresholdConfig `json:"disable"`
}

// ThresholdConfig defines enable/disable hysteresis parameters
type ThresholdConfig struct {
	Delay     time.Duration
	Threshold float64
}
