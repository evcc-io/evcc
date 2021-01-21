package vehicle

import "time"

type defaultConfig struct {
	Title    string
	Capacity int64
	User     string `validate:"required"`
	Password string `validate:"required" ui:",mask"`
	VIN      string
	Cache    time.Duration
}

func configDefaults() defaultConfig {
	return defaultConfig{
		Cache: interval,
	}
}
