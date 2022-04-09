package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util/request"
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
	chargerCmd.PersistentFlags().StringP(flagName, "n", "", fmt.Sprintf(flagNameDescription, "charger"))
	chargerCmd.PersistentFlags().IntP(flagCurrent, "I", noCurrent, flagCurrentDescription)
	chargerCmd.PersistentFlags().BoolP(flagEnable, "e", false, strings.Title(flagEnable))
	chargerCmd.PersistentFlags().BoolP(flagDisable, "d", false, strings.Title(flagDisable))
	chargerCmd.PersistentFlags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
	chargerCmd.PersistentFlags().Bool(flagHeaders, false, flagHeadersDescription)
}

func runCharger(cmd *cobra.Command, args []string) {
	log.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.Info("evcc %s", server.FormattedVersion())

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(conf); err != nil {
		log.Fatal(err)
	}

	// full http request log
	if cmd.PersistentFlags().Lookup(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// select single charger
	if err := selectByName(cmd, &conf.Chargers); err != nil {
		log.Fatal(err)
	}

	if err := cp.configureChargers(conf); err != nil {
		log.Fatal(err)
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
		if current != noCurrent {
			flagUsed = true

			if err := v.MaxCurrent(current); err != nil {
				log.Error("set current:", err)
			}
		}

		if cmd.PersistentFlags().Lookup(flagEnable).Changed {
			flagUsed = true

			if err := v.Enable(true); err != nil {
				log.Error("enable:", err)
			}
		}

		if cmd.PersistentFlags().Lookup(flagDisable).Changed {
			flagUsed = true

			if err := v.Enable(false); err != nil {
				log.Error("disable:", err)
			}
		}

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
