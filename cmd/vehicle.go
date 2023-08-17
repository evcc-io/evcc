package cmd

import (
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// vehicleCmd represents the vehicle command
var vehicleCmd = &cobra.Command{
	Use:   "vehicle [name]",
	Short: "Query configured vehicles",
	Args:  cobra.MaximumNArgs(1),
	Run:   runVehicle,
}

func init() {
	rootCmd.AddCommand(vehicleCmd)
	vehicleCmd.Flags().BoolP(flagStart, "a", false, flagStartDescription)
	vehicleCmd.Flags().BoolP(flagStop, "o", false, flagStopDescription)
	vehicleCmd.Flags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
	//lint:ignore SA1019 as Title is safe on ascii
	vehicleCmd.Flags().Bool(flagDiagnose, false, strings.Title(flagDiagnose))
}

func runVehicle(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		fatal(err)
	}

	// select single vehicle
	if err := selectByName(args, &conf.Vehicles); err != nil {
		fatal(err)
	}

	if err := configureVehicles(conf.Vehicles); err != nil {
		fatal(err)
	}

	vehicles := config.Vehicles().Devices()

	var flagUsed bool
	for _, v := range config.Instances(vehicles) {
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
		d := dumper{len: len(vehicles)}
		flag := cmd.Flags().Lookup(flagDiagnose).Changed

		for _, dev := range vehicles {
			v := dev.Instance()

			d.DumpWithHeader(dev.Config().Name, v)
			if flag {
				d.DumpDiagnosis(v)
			}
		}
	}

	// wait for shutdown
	<-shutdownDoneC()
}
