package main

//go:generate esc -o server/assets.go -pkg server -modtime 1566640112 -ignore .DS_Store dist

import (
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/cmd"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/server/config"
	"github.com/andig/evcc/vehicle"
)

func init() {
	// expose all configuration types to ui
	config.Add("charger", charger.Types())
	config.Add("meter", meter.Types())
	config.Add("vehicle", vehicle.Types())
}

func main() {
	cmd.Execute()
}
