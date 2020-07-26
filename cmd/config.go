package cmd

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/vehicle"
)

type config struct {
	URI        string
	Log        string
	Levels     map[string]string
	Interval   time.Duration
	Mqtt       provider.MqttConfig
	Influx     server.InfluxConfig
	Menu       []server.MenuConfig
	Messaging  messagingConfig
	Meters     []qualifiedConfig
	Chargers   []qualifiedConfig
	Vehicles   []qualifiedConfig
	Site       map[string]interface{}
	LoadPoints []map[string]interface{}
}

type qualifiedConfig struct {
	Name, Type string
	Other      map[string]interface{} `mapstructure:",remain"`
}

type typedConfig struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
}

type messagingConfig struct {
	Events   map[string]push.EventTemplate
	Services []typedConfig
}

// ConfigProvider provides configuration items
type ConfigProvider struct {
	meters   map[string]api.Meter
	chargers map[string]api.Charger
	vehicles map[string]api.Vehicle
}

// Meter provides meters by name
func (c *ConfigProvider) Meter(name string) api.Meter {
	if meter, ok := c.meters[name]; ok {
		return meter
	}
	log.FATAL.Fatalf("config: invalid meter %s", name)
	return nil
}

// Charger provides chargers by name
func (c *ConfigProvider) Charger(name string) api.Charger {
	if charger, ok := c.chargers[name]; ok {
		return charger
	}
	log.FATAL.Fatalf("config: invalid charger %s", name)
	return nil
}

// Vehicle provides vehicles by name
func (c *ConfigProvider) Vehicle(name string) api.Vehicle {
	if vehicle, ok := c.vehicles[name]; ok {
		return vehicle
	}
	log.FATAL.Fatalf("config: invalid vehicle %s", name)
	return nil
}

func (c *ConfigProvider) configure(conf config) {
	c.configureMeters(conf)
	c.configureChargers(conf)
	c.configureVehicles(conf)
}

func (c *ConfigProvider) configureMeters(conf config) {
	c.meters = make(map[string]api.Meter)
	for _, cc := range conf.Meters {
		c.meters[cc.Name] = meter.NewFromConfig(log, cc.Type, cc.Other)
	}
}

func (c *ConfigProvider) configureChargers(conf config) {
	c.chargers = make(map[string]api.Charger)
	for _, cc := range conf.Chargers {
		c.chargers[cc.Name] = charger.NewFromConfig(log, cc.Type, cc.Other)
	}
}

func (c *ConfigProvider) configureVehicles(conf config) {
	c.vehicles = make(map[string]api.Vehicle)
	for _, cc := range conf.Vehicles {
		c.vehicles[cc.Name] = vehicle.NewFromConfig(log, cc.Type, cc.Other)
	}
}
