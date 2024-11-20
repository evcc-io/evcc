package cmd

import (
	"github.com/spf13/cobra"
)

const (
	flagHeaders            = "log-headers"
	flagHeadersDescription = "Log headers"

	flagIgnoreDatabase            = "ignore-db"
	flagIgnoreDatabaseDescription = "Run command ignoring service database"

	flagDisableAuth            = "disable-auth"
	flagDisableAuthDescription = "Disable authentication (dangerous)"

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

	flagReset            = "reset"
	flagResetDescription = "Reset migrated settings"

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

	flagRepeat            = "repeat"
	flagRepeatDescription = "Repeat until interrupted"

	flagDigits = "digits"
	flagDelay  = "delay"
	flagForce  = "force"
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
