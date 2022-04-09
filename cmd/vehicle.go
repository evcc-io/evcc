package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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
	vehicleCmd.PersistentFlags().StringP(flagName, "n", "", fmt.Sprintf(flagNameDescription, "vehicle"))
	vehicleCmd.PersistentFlags().BoolP(flagStart, "a", false, flagStartDescription)
	vehicleCmd.PersistentFlags().BoolP(flagStop, "o", false, flagStopDescription)
	vehicleCmd.PersistentFlags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
	vehicleCmd.PersistentFlags().Bool(flagHeaders, false, flagHeadersDescription)
}

func runVehicle(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.Info("evcc %s", server.FormattedVersion())

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// full http request log
	if cmd.PersistentFlags().Lookup(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// select single vehicle
	if err := selectByName(cmd, &conf.Vehicles); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := cp.configureVehicles(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	vehicles := cp.vehicles
	if len(args) == 1 {
		arg := args[0]
		vehicles = map[string]api.Vehicle{arg: cp.Vehicle(arg)}
	}

	d := dumper{len: len(vehicles)}

	var flagUsed bool
	for _, v := range vehicles {
		if cmd.PersistentFlags().Lookup(flagWakeup).Changed {
			flagUsed = true

			if vv, ok := v.(api.AlarmClock); ok {
				if err := vv.WakeUp(); err != nil {
					log.Error("wakeup:", err)
				}
			} else {
				log.Error("wakeup: not implemented")
			}
		}

		if cmd.PersistentFlags().Lookup(flagStart).Changed {
			flagUsed = true

			if vv, ok := v.(api.VehicleChargeController); ok {
				if err := vv.StartCharge(); err != nil {
					log.Error("start charge:", err)
				}
			} else {
				log.Error("start charge: not implemented")
			}
		}

		if cmd.PersistentFlags().Lookup(flagStop).Changed {
			flagUsed = true

			if vv, ok := v.(api.VehicleChargeController); ok {
				if err := vv.StopCharge(); err != nil {
					log.Error("stop charge:", err)
				}
			} else {
				log.Error("stop charge: not implemented")
			}
		}
	}

	if !flagUsed {
		for name, v := range vehicles {
			d.DumpWithHeader(name, v)
		}
	}
}
