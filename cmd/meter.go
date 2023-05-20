package cmd

import (
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// meterCmd represents the meter command
var meterCmd = &cobra.Command{
	Use:       "meter [name]",
	Short:     "Query configured meters",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"name"},
	Run:       runMeter,
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

	meters := config.MetersMap()

	d := dumper{len: len(meters)}
	for name, v := range meters {
		d.DumpWithHeader(name, v)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
