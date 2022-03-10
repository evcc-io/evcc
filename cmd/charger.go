package cmd

import (
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// chargerCmd represents the charger command
var chargerCmd = &cobra.Command{
	Use:   "charger [name]",
	Short: "Query configured chargers",
	Run:   runCharger,
}

func init() {
	rootCmd.AddCommand(chargerCmd)
	chargerCmd.PersistentFlags().StringP("name", "n", "", "select charger by name")
	chargerCmd.PersistentFlags().IntP("current", "I", -1, "set current")
	chargerCmd.PersistentFlags().BoolP("enable", "e", false, "enable")
	chargerCmd.PersistentFlags().BoolP("disable", "d", false, "disable")
}

func runCharger(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// select single charger
	if name := cmd.PersistentFlags().Lookup("name").Value.String(); name != "" {
		for _, cfg := range conf.Chargers {
			if cfg.Name == name {
				conf.Chargers = []qualifiedConfig{cfg}
				break
			}
		}
	}

	if err := cp.configureChargers(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	stopC := make(chan struct{})
	go shutdown.Run(stopC)

	chargers := cp.chargers
	if len(args) == 1 {
		arg := args[0]
		chargers = map[string]api.Charger{arg: cp.Charger(arg)}
	}

	var current int64
	if flag := cmd.PersistentFlags().Lookup("current").Value.String(); flag != "-1" {
		var err error
		current, err = strconv.ParseInt(flag, 10, 64)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	d := dumper{len: len(chargers)}
	for name, v := range chargers {
		if current >= 0 {
			fmt.Println("Set current:", current)
			if err := v.MaxCurrent(current); err != nil {
				log.ERROR.Println(err)
			}
		}

		if flag := cmd.PersistentFlags().Lookup("enable").Value.String(); flag == "true" {
			fmt.Println("Set enabled")
			if err := v.Enable(true); err != nil {
				log.ERROR.Println(err)
			}
		}

		if flag := cmd.PersistentFlags().Lookup("disable").Value.String(); flag == "true" {
			fmt.Println("Set disabled")
			if err := v.Enable(false); err != nil {
				log.ERROR.Println(err)
			}
		}

		d.DumpWithHeader(name, v)
	}

	close(stopC)
	<-shutdown.Done()
}
