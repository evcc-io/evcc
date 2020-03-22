package cmd

import (
	"time"

	"github.com/andig/evcc/api"
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
	Chargers   []chargerConfig
	SoCs       []socConfig
	LoadPoints []loadPointConfig
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

type providerConfig struct {
	Type       string
	Topic      string
	Cmd        string
	Multiplier float64
	Timeout    time.Duration
	Cache      time.Duration
}

type meterConfig struct {
	Name   string
	Type   string
	Power  *providerConfig
	Energy *providerConfig
}

type chargerConfig struct {
	Name string
	Type string

	// wallbe charger
	URI string

	// composite charger
	Status     *providerConfig // Charger
	MaxCurrent *providerConfig // ChargeController
	Enable     *providerConfig // Charger
	Enabled    *providerConfig // Charger
}

type socConfig struct {
	Name     string
	Title    string
	Type     string
	Capacity int64
	Charge   *providerConfig
}

type loadPointConfig struct {
	Name          string
	GridMeter     string // api.Meter
	PVMeter       string // api.Meter
	ChargeMeter   string // api.Meter
	Charger       string // api.Charger
	SoC           string // api.SoC
	Mode          api.ChargeMode
	Phases        int64
	MinCurrent    int64
	MaxCurrent    int64
	Steepness     int64
	GuardDuration time.Duration
	Voltage       float64
	ResidualPower float64
}
