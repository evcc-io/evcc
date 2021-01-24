package main

//go:generate esc -o server/assets.go -pkg server -modtime 1566640112 -ignore .DS_Store dist

import (
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/cmd"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server/config"
	"github.com/andig/evcc/vehicle"
)

func init() {
	// expose all configuration types to ui
	config.SetTypes("charger", charger.Types())
	config.SetTypes("meter", meter.Types())
	config.SetTypes("vehicle", vehicle.Types())
	config.SetTypes("plugin", provider.Types())
}

func main() {
	cmd.Execute()
}
