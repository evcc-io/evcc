package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
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
	meterCmd.PersistentFlags().StringP(flagName, "n", "", fmt.Sprintf(flagNameDescription, "meter"))
}

func runMeter(cmd *cobra.Command, args []string) {
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

	// select single meter
	if err := selectByName(cmd, &conf.Meters); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := cp.configureMeters(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	meters := cp.meters
	if len(args) == 1 {
		name := args[0]
		meter, err := cp.Meter(name)
		if err != nil {
			log.FATAL.Fatal(err)
		}
		meters = map[string]api.Meter{name: meter}
	}

	d := dumper{len: len(meters)}
	for name, v := range meters {
		d.DumpWithHeader(name, v)
	}
}
