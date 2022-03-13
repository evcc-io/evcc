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

const (
	pCurrent = "current"
	pEnable  = "enable"
	pDisable = "disable"
	pWakeup  = "wakeup"
)

func init() {
	rootCmd.AddCommand(chargerCmd)
	chargerCmd.PersistentFlags().StringP("name", "n", "", "select charger by name")
	chargerCmd.PersistentFlags().IntP(pCurrent, "I", -1, "set current")
	chargerCmd.PersistentFlags().BoolP(pEnable, "e", false, pEnable)
	chargerCmd.PersistentFlags().BoolP(pDisable, "d", false, pDisable)
	chargerCmd.PersistentFlags().BoolP(pWakeup, "w", false, pWakeup)
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
	if flag := cmd.PersistentFlags().Lookup(pCurrent).Value.String(); flag != "-1" {
		var err error
		current, err = strconv.ParseInt(flag, 10, 64)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	d := dumper{len: len(chargers)}
	for name, v := range chargers {
		if current >= 0 {
			if err := v.MaxCurrent(current); err != nil {
				log.ERROR.Println("set current:", err)
			}
		}

		if flag := cmd.PersistentFlags().Lookup(pEnable).Value.String(); flag == "true" {
			if err := v.Enable(true); err != nil {
				log.ERROR.Println("enable:", err)
			}
		}

		if flag := cmd.PersistentFlags().Lookup(pDisable).Value.String(); flag == "true" {
			if err := v.Enable(false); err != nil {
				log.ERROR.Println("disable:", err)
			}
		}

		if flag := cmd.PersistentFlags().Lookup(pWakeup).Value.String(); flag == "true" {
			if vv, ok := v.(api.AlarmClock); ok {
				if err := vv.WakeUp(); err != nil {
					log.ERROR.Println("wakeup:", err)
				}
			} else {
				log.ERROR.Println("wakeup: not implemented")
			}
		}

		d.DumpWithHeader(name, v)
	}

	close(stopC)
	<-shutdown.Done()
}
