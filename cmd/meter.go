package cmd

import (
	"fmt"

	"github.com/mark-sch/evcc/api"
	"github.com/mark-sch/evcc/server"
	"github.com/mark-sch/evcc/util"
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
	rootCmd.AddCommand(meterCmd)
}

func runMeter(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if conf.Mqtt.Broker != "" {
		configureMQTT(conf.Mqtt)
	}

	cp := &ConfigProvider{}
	cp.configureMeters(conf)

	meters := cp.meters
	if len(args) == 1 {
		arg := args[0]
		meters = map[string]api.Meter{arg: cp.Meter(arg)}
	}

	for name, v := range meters {
		if len(meters) != 1 {
			fmt.Println(name)
		}

		dumpAPIs(v)
	}
}
