package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	flagHeaders            = "log-headers"
	flagHeadersDescription = "Log headers"

	flagName            = "name"
	flagNameDescription = "Select %s by name"

	flagCurrent            = "current"
	flagCurrentDescription = "Set maximum current"

	flagEnable  = "enable"
	flagDisable = "disable"

	flagWakeup            = "wakeup"
	flagWakeupDescription = "Wake up"

	flagStart            = "start"
	flagStartDescription = "Start charging"

	flagStop            = "stop"
	flagStopDescription = "Stop charging"

	flagDigits = "digits"
)

func selectByName(cmd *cobra.Command, conf *[]qualifiedConfig) error {
	flag := cmd.PersistentFlags().Lookup(flagName)
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
