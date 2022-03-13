package cmd

import (
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

const noCurrent = -1

func init() {
	rootCmd.AddCommand(chargerCmd)
	chargerCmd.PersistentFlags().StringP(flagName, "n", "", "select charger by name")
	chargerCmd.PersistentFlags().IntP(flagCurrent, "I", noCurrent, "set current")
	chargerCmd.PersistentFlags().BoolP(flagEnable, "e", false, flagEnable)
	chargerCmd.PersistentFlags().BoolP(flagDisable, "d", false, flagDisable)
	chargerCmd.PersistentFlags().BoolP(flagWakeup, "w", false, flagWakeup)
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
	if name := cmd.PersistentFlags().Lookup(flagName).Value.String(); name != "" {
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

	current := int64(noCurrent)
	if flag := cmd.PersistentFlags().Lookup(flagCurrent); flag.Changed {
		var err error
		current, err = strconv.ParseInt(flag.Value.String(), 10, 64)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	var flagUsed bool
	for _, v := range chargers {
		if current >= 0 {
			flagUsed = true

			if err := v.MaxCurrent(current); err != nil {
				log.ERROR.Println("set current:", err)
			}
		}

		if cmd.PersistentFlags().Lookup(flagEnable).Changed {
			flagUsed = true

			if err := v.Enable(true); err != nil {
				log.ERROR.Println("enable:", err)
			}
		}

		if cmd.PersistentFlags().Lookup(flagDisable).Changed {
			flagUsed = true

			if err := v.Enable(false); err != nil {
				log.ERROR.Println("disable:", err)
			}
		}

		if cmd.PersistentFlags().Lookup(flagWakeup).Changed {
			flagUsed = true

			if vv, ok := v.(api.AlarmClock); ok {
				if err := vv.WakeUp(); err != nil {
					log.ERROR.Println("wakeup:", err)
				}
			} else {
				log.ERROR.Println("wakeup: not implemented")
			}
		}
	}

	if !flagUsed {
		d := dumper{len: len(chargers)}
		for name, v := range chargers {
			d.DumpWithHeader(name, v)
		}
	}

	close(stopC)
	<-shutdown.Done()
}
