package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/util/request"
)

type Config struct {
	Embed    `mapstructure:",squash"`
	User     string `validate:"required"`
	Password string `validate:"required" ui:",mask"`
	VIN      string
	Cache    time.Duration
}

func defaults() Config {
	return Config{
		Cache: interval,
	}
}

type ConfigWithTimeout struct {
	Config  `mapstructure:",squash"`
	Timeout time.Duration
}

func (c Config) WithTimeout() ConfigWithTimeout {
	return ConfigWithTimeout{
		Config:  defaults(),
		Timeout: request.Timeout,
	}
}

type Embed struct {
	Title_      string `mapstructure:"title" ui:"de=Anzeigename"`
	Capacity_   int64  `mapstructure:"capacity" ui:"de=Kapazität (kWh)"`
	Identifier_ string `mapstructure:"identifier" ui:"de=Identifikation"`
}

// Title implements the api.Vehicle interface
func (v *Embed) Title() string {
	return v.Title_
}

// Capacity implements the api.Vehicle interface
func (v *Embed) Capacity() int64 {
	return v.Capacity_
}

// Identify implements the api.Identifier interface
func (v *Embed) Identify() (string, error) {
	return v.Identifier_, nil
}
