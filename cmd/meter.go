package cmd

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// meterCmd represents the meter command
var meterCmd = &cobra.Command{
	Use:   "meter [name]",
	Short: "Query configured meters",
	Args:  cobra.MaximumNArgs(1),
	Run:   runMeter,
}

func init() {
	rootCmd.AddCommand(meterCmd)
	meterCmd.Flags().StringP(flagBatteryMode, "b", "", flagBatteryModeDescription)
}

func runMeter(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// select single meter
	if err := selectByName(args, &conf.Meters); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureMeters(conf.Meters); err != nil {
		log.FATAL.Fatal(err)
	}

	mode := api.BatteryUnknown
	if val := cmd.Flags().Lookup(flagBatteryMode).Value.String(); val != "" {
		var err error
		mode, err = api.BatteryModeString(val)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	meters := config.Meters().Devices()

	if mode != api.BatteryUnknown {
		for _, dev := range meters {
			v := dev.Instance()
			if b, ok := v.(api.BatteryController); ok {
				if err := b.SetBatteryMode(mode); err != nil {
					log.FATAL.Fatalln("set battery mode:", err)
				}
			}
		}
	}

	d := dumper{len: len(meters)}
	for _, dev := range meters {
		v := dev.Instance()

		d.DumpWithHeader(dev.Config().Name, v)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
