package cmd

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/vehicle"
)

type ConfigProvider struct {
	meters   map[string]api.Meter
	chargers map[string]api.Charger
	vehicles map[string]api.Vehicle
}

func (c *ConfigProvider) Meter(name string) api.Meter {
	if meter, ok := c.meters[name]; ok {
		return meter
	}
	log.FATAL.Fatalf("config: invalid meter %s", name)
	return nil
}

func (c *ConfigProvider) Charger(name string) api.Charger {
	if charger, ok := c.chargers[name]; ok {
		return charger
	}
	log.FATAL.Fatalf("config: invalid charger %s", name)
	return nil
}

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
		c.meters[cc.Name] = core.NewMeterFromConfig(log, cc.Other)
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
