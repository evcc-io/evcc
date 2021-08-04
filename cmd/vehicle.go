package cmd

import (
	"errors"
	"fmt"
	"time"

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
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(conf); err != nil {
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
NEXT:
	for name, v := range vehicles {
		start := time.Now()

	WAIT:
		// wait up to 1m for the vehicle to wakeup
		for {
			if time.Since(start) > time.Minute {
				log.ERROR.Println(api.ErrTimeout)
				continue NEXT
			}

			if _, err := v.SoC(); err != nil {
				if errors.Is(err, api.ErrMustRetry) {
					time.Sleep(5 * time.Second)
					fmt.Print(".")
					continue WAIT
				}

				log.ERROR.Println(err)
				continue NEXT
			}

			break
		}

		d.DumpWithHeader(name, v)
	}
}
