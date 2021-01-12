package vehicle

import "time"

type defaultConfig struct {
	Title          string
	Capacity       int64
	User, Password string `validate:"required"`
	VIN            string
	Cache          time.Duration
}

func configDefaults() defaultConfig {
	return defaultConfig{
		Cache: interval,
	}
}
