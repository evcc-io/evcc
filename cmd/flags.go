package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagHeaders            = "log-headers"
	flagHeadersDescription = "Log headers"

	flagBatteryMode                = "battery-mode"
	flagBatteryModeDescription     = "Set battery mode (normal, hold, charge)"
	flagBatteryModeWait            = "battery-mode-wait"
	flagBatteryModeWaitDescription = "Wait given duration during which potential watchdogs are active"

	flagCurrent            = "current"
	flagCurrentDescription = "Set maximum current"

	flagPhases            = "phases"
	flagPhasesDescription = "Set usable phases (1 or 3)"

	flagCloud            = "cloud"
	flagCloudDescription = "Use cloud service (requires sponsor token)"

	flagEnable  = "enable"
	flagDisable = "disable"

	flagDiagnose            = "diagnose"
	flagDiagnoseDescription = "Diagnose"

	flagWakeup            = "wakeup"
	flagWakeupDescription = "Wake up"

	flagStart            = "start"
	flagStartDescription = "Start charging"

	flagStop            = "stop"
	flagStopDescription = "Stop charging"

	flagDigits = "digits"
	flagDelay  = "delay"
)

func bind(cmd *cobra.Command, key string, flagName ...string) {
	name := key
	if len(flagName) == 1 {
		name = flagName[0]
	}
	if err := viper.BindPFlag(key, cmd.Flags().Lookup(name)); err != nil {
		panic(err)
	}
}

func bindP(cmd *cobra.Command, key string, flagName ...string) {
	name := key
	if len(flagName) == 1 {
		name = flagName[0]
	}
	if err := viper.BindPFlag(key, cmd.PersistentFlags().Lookup(name)); err != nil {
		panic(err)
	}
}

func selectByName(args []string, conf *[]config.Named) error {
	if len(args) != 1 {
		return nil
	}

	name := args[0]

	for _, c := range *conf {
		if c.Name == name {
			*conf = []config.Named{c}
			return nil
		}
	}

	return fmt.Errorf("%s not found", name)
}
