package cmd

import (
	"fmt"

	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
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
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(vehicleCmd)
	configureCommand(vehicleCmd)
}

func runVehicle(cmd *cobra.Command, args []string) {
	level, _ := cmd.PersistentFlags().GetString("log")
	configureLogging(level)
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	vehicles := configureVehicles(conf)

	for name, v := range vehicles {
		if len(args) == 1 {
			if target := args[0]; name != target {
				if _, ok := vehicles[target]; !ok {
					log.FATAL.Fatalf("charger not found: %s", target)
				}
				continue
			}
		} else if len(vehicles) != 1 {
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
