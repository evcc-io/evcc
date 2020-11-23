package cmd

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
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
	if conf.Mqtt.Broker != "" {
		configureMQTT(conf.Mqtt)
	}

	if err := cp.configureVehicles(conf); err != nil {
		cp.Close() // cleanup any open sessions
		log.FATAL.Fatal(err)
	}

	defer cp.Close() // cleanup on exit

	vehicles := cp.vehicles
	if len(args) == 1 {
		arg := args[0]
		vehicles = map[string]api.Vehicle{arg: cp.Vehicle(arg)}
	}

	w := dumpFormat()

	for name, v := range vehicles {
		if len(vehicles) != 1 {
			fmt.Fprintln(w, name)
			fmt.Fprintln(w, strings.Repeat("-", len(name)))
		}

		if soc, err := v.ChargeState(); err != nil {
			fmt.Printf("State: %v\n", err)
		} else {
			fmt.Printf("State: %.0f%%\n", soc)
		}

		dumpAPIs(w, v)

		if len(vehicles) != 1 {
			fmt.Fprintln(w)
		}

		w.Flush()
	}
}
