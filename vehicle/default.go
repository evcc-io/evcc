package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/util/request"
)

type config struct {
	embed    `mapstructure:",squash"`
	User     string `validate:"required"`
	Password string `validate:"required" ui:",mask"`
	VIN      string
	Cache    time.Duration
}

func defaults() config {
	return config{
		Cache: interval,
	}
}

type configWithTimeout struct {
	config  `mapstructure:",embed"`
	Timeout time.Duration
}

func (c config) WithTimeout() configWithTimeout {
	return configWithTimeout{
		config:  defaults(),
		Timeout: request.Timeout,
	}
}

// func defaultsWithTimeout() configWithTimeout {
// 	return configWithTimeout{
// 		config:  defaults(),
// 		Timeout: request.Timeout,
// 	}
// }

type embed struct {
	Title_      string `mapstructure:"title"`
	Capacity_   int64  `mapstructure:"capacity"`
	Identifier_ string `mapstructure:"identifier"`
}

// Title implements the api.Vehicle interface
func (v *embed) Title() string {
	return v.Title_
}

// Capacity implements the api.Vehicle interface
func (v *embed) Capacity() int64 {
	return v.Capacity_
}

// Identify implements the api.Identifier interface
func (v *embed) Identify() (string, error) {
	return v.Identifier_, nil
}
