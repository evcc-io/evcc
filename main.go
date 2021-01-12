package main

//go:generate esc -o server/assets.go -pkg server -modtime 1566640112 -ignore .DS_Store dist

import (
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/cmd"
	"github.com/andig/evcc/server/config"
	"github.com/andig/evcc/vehicle"
)

func init() {
	config.Add("charger", charger.ConfigTypes())
	config.Add("vehicle", vehicle.ConfigTypes())
}

func main() {
	cmd.Execute()
}
