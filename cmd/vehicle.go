package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
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
	vehicleCmd.Flags().BoolP(flagStart, "a", false, flagStartDescription)
	vehicleCmd.Flags().BoolP(flagStop, "o", false, flagStopDescription)
	vehicleCmd.Flags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
}

func runVehicle(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
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
		name := args[0]
		vehicle, err := cp.Vehicle(name)
		if err != nil {
			log.FATAL.Fatal(err)
		}
		vehicles = map[string]api.Vehicle{name: vehicle}
	}

	d := dumper{len: len(vehicles)}

	var flagUsed bool
	for _, v := range vehicles {
		if cmd.Flags().Lookup(flagWakeup).Changed {
			flagUsed = true

			if vv, ok := v.(api.Resurrector); ok {
				if err := vv.WakeUp(); err != nil {
					log.ERROR.Println("wakeup:", err)
				}
			} else {
				log.ERROR.Println("wakeup: not implemented")
			}
		}

		if cmd.Flags().Lookup(flagStart).Changed {
			flagUsed = true

			if vv, ok := v.(api.VehicleChargeController); ok {
				if err := vv.StartCharge(); err != nil {
					log.ERROR.Println("start charge:", err)
				}
			} else {
				log.ERROR.Println("start charge: not implemented")
			}
		}

		if cmd.Flags().Lookup(flagStop).Changed {
			flagUsed = true

			if vv, ok := v.(api.VehicleChargeController); ok {
				if err := vv.StopCharge(); err != nil {
					log.ERROR.Println("stop charge:", err)
				}
			} else {
				log.ERROR.Println("stop charge: not implemented")
			}
		}
	}

	if !flagUsed {
		for name, v := range vehicles {
			d.DumpWithHeader(name, v)
		}
	}
}
