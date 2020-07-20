package cmd

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// vehicleCmd represents the vehicle command
var vehicleCmd = &cobra.Command{
	Use:   "vehicle [name]",
	Short: "Query configured vehicles",
	Run:   runVehicle,
}

func init() {
	rootCmd.AddCommand(vehicleCmd)
}

func runVehicle(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	cp := &ConfigProvider{}
	cp.configureVehicles(conf)

	vehicles := cp.vehicles
	if len(args) == 1 {
		arg := args[0]
		vehicles = map[string]api.Vehicle{arg: cp.Vehicle(arg)}
	}

	for name, v := range vehicles {
		if len(vehicles) != 1 {
			fmt.Println(name)
		}

		if soc, err := v.ChargeState(); err != nil {
			fmt.Printf("State: %v\n", err)
		} else {
			fmt.Printf("State: %.0f%%\n", soc)
		}

		dumpAPIs(v)
	}
}
