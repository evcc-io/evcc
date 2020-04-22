package cmd

import (
	"fmt"

	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// meterCmd represents the meter command
var meterCmd = &cobra.Command{
	Use:   "meter [name]",
	Short: "Query configured meters",
	Run:   runMeter,
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(meterCmd)
	configureCommand(meterCmd)
}

func runMeter(cmd *cobra.Command, args []string) {
	level, _ := cmd.PersistentFlags().GetString("log")
	configureLogging(level)
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	cp := &ConfigProvider{}
	cp.configureMeters(conf)

	for name, v := range cp.meters {
		if len(args) == 1 {
			if target := args[0]; name != target {
				_ = cp.Meter(target)
				continue
			}
		} else if len(cp.meters) != 1 {
			fmt.Println(name)
		}

		dumpAPIs(v)
	}
}
