package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagSqlite            = "sqlite"
	flagSqliteDescription = "Sqlite database file"

	flagHeaders            = "log-headers"
	flagHeadersDescription = "Log headers"

	flagName            = "name"
	flagNameDescription = "Select %s by name"

	flagCurrent            = "current"
	flagCurrentDescription = "Set maximum current"

	flagPhases            = "phases"
	flagPhasesDescription = "Set usable phases (1 or 3)"

	flagEnable  = "enable"
	flagDisable = "disable"

	flagWakeup            = "wakeup"
	flagWakeupDescription = "Wake up"

	flagStart            = "start"
	flagStartDescription = "Start charging"

	flagStop            = "stop"
	flagStopDescription = "Stop charging"

	flagDigits = "digits"
	flagDelay  = "delay"
)

func bind(cmd *cobra.Command, flag string) {
	if err := viper.BindPFlag(flag, cmd.Flags().Lookup(flag)); err != nil {
		panic(err)
	}
}

func bindP(cmd *cobra.Command, flag string) {
	if err := viper.BindPFlag(flag, cmd.PersistentFlags().Lookup(flag)); err != nil {
		panic(err)
	}
}

func selectByName(cmd *cobra.Command, conf *[]qualifiedConfig) error {
	flag := cmd.Flags().Lookup(flagName)
	if !flag.Changed {
		return nil
	}

	name := flag.Value.String()

	for _, cfg := range *conf {
		if cfg.Name == name {
			*conf = []qualifiedConfig{cfg}
			return nil
		}
	}

	return fmt.Errorf("%s not found", name)
}
