package cmd

import (
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// chargerCmd represents the charger command
var chargerCmd = &cobra.Command{
	Use:   "charger [name]",
	Short: "Query configured chargers",
	Args:  cobra.MaximumNArgs(1),
	Run:   runCharger,
}

func init() {
	rootCmd.AddCommand(chargerCmd)
	chargerCmd.Flags().Float64P(flagCurrent, "i", 0, flagCurrentDescription)
	//lint:ignore SA1019 as Title is safe on ascii
	chargerCmd.Flags().BoolP(flagEnable, "e", false, strings.Title(flagEnable))
	//lint:ignore SA1019 as Title is safe on ascii
	chargerCmd.Flags().BoolP(flagDisable, "d", false, strings.Title(flagDisable))
	chargerCmd.Flags().Bool(flagDiagnose, false, flagDiagnoseDescription)
	chargerCmd.Flags().BoolP(flagWakeup, "w", false, flagWakeupDescription)
	chargerCmd.Flags().IntP(flagPhases, "p", 0, flagPhasesDescription)
	chargerCmd.Flags().Bool(flagHeartbeat, false, flagHeartbeatDescription)
	chargerCmd.Flags().Duration(flagTimeout, time.Second, flagTimeoutDescription)
}

func runCharger(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureChargers(conf.Chargers, args...); err != nil {
		log.FATAL.Fatal(err)
	}

	var phases int
	if flag := cmd.Flag(flagPhases); flag.Changed {
		var err error
		phases, err = strconv.Atoi(flag.Value.String())
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	chargers := config.Chargers().Devices()

	var flagUsed bool
	for _, v := range config.Instances(chargers) {
		if flag := cmd.Flag(flagCurrent); flag.Changed {
			flagUsed = true

			current, err := strconv.ParseFloat(flag.Value.String(), 64)
			if err != nil {
				log.ERROR.Fatalln(err)
			}

			if vv, ok := v.(api.ChargerEx); ok {
				if err := vv.MaxCurrentMillis(current); err != nil {
					log.ERROR.Println("set current:", err)
				}
			} else {
				if err := v.MaxCurrent(int64(current)); err != nil {
					log.ERROR.Println("set current:", err)
				}
			}
		}

		if cmd.Flag(flagEnable).Changed {
			flagUsed = true

			if err := v.Enable(true); err != nil {
				log.ERROR.Println("enable:", err)
			}
		}

		if cmd.Flag(flagDisable).Changed {
			flagUsed = true

			if err := v.Enable(false); err != nil {
				log.ERROR.Println("disable:", err)
			}
		}

		if cmd.Flag(flagWakeup).Changed {
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

	if ok, _ := cmd.Flags().GetBool(flagHeartbeat); flagUsed && ok {
		log.INFO.Println("running heartbeat until interrupted (Ctrl-C to stop)")
		time.Sleep(time.Hour)
	}

	if !flagUsed {
		timeout, _ := cmd.Flags().GetDuration(flagTimeout)
		d := dumper{len: len(chargers), timeout: timeout}
		flag := cmd.Flag(flagDiagnose).Changed

		for _, dev := range chargers {
			v := dev.Instance()

			d.DumpWithHeader(deviceHeader(dev), v)
			if flag {
				d.DumpDiagnosis(v)
			}
		}
	} else if ok, _ := cmd.Flags().GetBool(flagHeartbeat); ok {
		log.INFO.Println("running heartbeat (if any) until interrupted (Ctrl-C to stop)")
		time.Sleep(time.Hour)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
