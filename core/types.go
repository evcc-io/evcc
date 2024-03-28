package core

import "time"

// PollConfig defines the vehicle polling mode and interval
type PollConfig struct {
	Mode     string        `mapstructure:"mode"`     // polling mode charging (default), connected, always
	Interval time.Duration `mapstructure:"interval"` // interval when not charging
}

// SocConfig defines soc settings, estimation and update behavior
type SocConfig struct {
	Poll     PollConfig `mapstructure:"poll"`
	Estimate *bool      `mapstructure:"estimate"`
}

// Task is the task type
type Task = func()
