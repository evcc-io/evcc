package cmd

import (
	"time"

	"github.com/andig/evcc/core"
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
	Messaging  messagingConfig
	Meters     []namedConfig
	Chargers   []typedConfig
	Vehicles   []typedConfig
	LoadPoints []core.Config
}

type namedConfig struct {
	Name  string
	Other map[string]interface{} `mapstructure:",remain"`
}

type typedConfig struct {
	Name, Type string
	Other      map[string]interface{} `mapstructure:",remain"`
}

type messagingConfig struct {
	Events   map[string]push.EventTemplate
	Services []messagingService
}

type messagingService struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
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
