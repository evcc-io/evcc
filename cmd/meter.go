package cmd

import (
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

	meters := config.Meters().Devices()

	d := dumper{len: len(meters)}
	for _, dev := range meters {
		v := dev.Instance()

		d.DumpWithHeader(dev.Config().Name, v)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
