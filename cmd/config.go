package cmd

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/server"
)

type config struct {
	URI        string
	Log        string
	Interval   time.Duration
	Mqtt       mqttConfig
	Influx     influxConfig
	Menu       []server.MenuConfig
	Pushover   messagingConfig
	Meters     []meterConfig
	Chargers   []typedConfig
	Vehicles   []typedConfig
	LoadPoints []loadPointConfig
}

type typedConfig struct {
	Name, Type string
	Other      map[string]interface{} `mapstructure:",remain"`
}

type messagingConfig struct {
	App        string
	Recipients []string
	Events     map[string]push.EventTemplate
}

type mqttConfig struct {
	Broker   string
	User     string
	Password string
}

type influxConfig struct {
	URL      string
	Database string
	User     string
	Password string
	Interval time.Duration
}

type meterConfig struct {
	Name   string
	Type   string
	Power  *provider.Config
	Energy *provider.Config
}

type loadPointConfig struct {
	Name          string
	GridMeter     string // api.Meter
	PVMeter       string // api.Meter
	ChargeMeter   string // api.Meter
	Charger       string // api.Charger
	Vehicle       string // api.Vehicle
	Mode          api.ChargeMode
	Phases        int64
	MinCurrent    int64
	MaxCurrent    int64
	Steepness     int64
	GuardDuration time.Duration
	Voltage       float64
	ResidualPower float64
}
