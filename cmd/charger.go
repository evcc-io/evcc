package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/spf13/cobra"
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
	chargerCmd.Flags().IntP(flagCurrent, "i", noCurrent, flagCurrentDescription)
	//lint:ignore SA1019 as Title is safe on ascii
	chargerCmd.Flags().BoolP(flagEnable, "e", false, strings.Title(flagEnable))
	//lint:ignore SA1019 as Title is safe on ascii
	chargerCmd.Flags().BoolP(flagDisable, "d", false, strings.Title(flagDisable))
	//lint:ignore SA1019 as Title is safe on ascii
	chargerCmd.Flags().Bool(flagDiagnose, false, strings.Title(flagDiagnose))
	chargerCmd.Flags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
	chargerCmd.Flags().IntP(flagPhases, "p", 0, flagPhasesDescription)
}

func runCharger(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// select single charger
	if err := selectByName(cmd, &conf.Chargers); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := cp.configureChargers(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	chargers := cp.chargers
	if len(args) == 1 {
		name := args[0]
		charger, err := cp.Charger(name)
		if err != nil {
			log.FATAL.Fatal(err)
		}
		chargers = map[string]api.Charger{name: charger}
	}

	current := int64(noCurrent)
	if flag := cmd.Flags().Lookup(flagCurrent); flag.Changed {
		var err error
		current, err = strconv.ParseInt(flag.Value.String(), 10, 64)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	var phases int
	if flag := cmd.Flags().Lookup(flagPhases); flag.Changed {
		var err error
		phases, err = strconv.Atoi(flag.Value.String())
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	var flagUsed bool
	for _, v := range chargers {
		if current != noCurrent {
			flagUsed = true

			if err := v.MaxCurrent(current); err != nil {
				log.ERROR.Println("set current:", err)
			}
		}

		if cmd.Flags().Lookup(flagEnable).Changed {
			flagUsed = true

			if err := v.Enable(true); err != nil {
				log.ERROR.Println("enable:", err)
			}
		}

		if cmd.Flags().Lookup(flagDisable).Changed {
			flagUsed = true

			if err := v.Enable(false); err != nil {
				log.ERROR.Println("disable:", err)
			}
		}

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

		if phases > 0 {
			flagUsed = true

			if vv, ok := v.(api.PhaseSwitcher); ok {
				if err := vv.Phases1p3p(phases); err != nil {
					log.ERROR.Println("set phases:", err)
				}
			} else {
				log.ERROR.Println("phases: not implemented")
			}
		}
	}

	if !flagUsed {
		d := dumper{len: len(chargers)}
		flag := cmd.Flags().Lookup(flagDiagnose).Changed

		for name, v := range chargers {
			d.DumpWithHeader(name, v)
			if flag {
				d.DumpDiagnosis(v)
			}
		}
	}

	// wait for shutdown
	<-shutdownDoneC()
}
