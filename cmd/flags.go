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

	flagCurrent            = "current"
	flagCurrentDescription = "Set maximum current"

	flagPhases            = "phases"
	flagPhasesDescription = "Set usable phases (1 or 3)"

	flagEnable   = "enable"
	flagDisable  = "disable"
	flagDiagnose = "diagnose"

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
