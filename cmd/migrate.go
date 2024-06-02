package cmd

import (
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate yaml to database (overwrites db settings)",
	Args:  cobra.ExactArgs(0),
	Run:   runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolP(flagReset, "c", false, flagResetDescription)
}

func runMigrate(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	reset := cmd.Flags().Lookup(flagReset).Changed

	// TODO remove yaml file
	if reset {
		settings.Delete(keys.Interval)
		settings.Delete(keys.SponsorToken)
	} else {
		log.DEBUG.Println("migrate global settings")
		settings.SetInt(keys.Interval, int64(conf.Interval))
		settings.SetString(keys.SponsorToken, conf.SponsorToken)
	}

	if reset {
		settings.Delete(keys.Network)
	} else {
		log.DEBUG.Println("migrate network")
		_ = settings.SetJson(keys.Network, conf)
	}

	if reset {
		settings.Delete(keys.Mqtt)
	} else {
		log.DEBUG.Println("migrate mqtt")
		_ = settings.SetJson(keys.Mqtt, conf)
	}

	if reset {
		settings.Delete(keys.Influx)
	} else {
		log.DEBUG.Println("migrate influx")
		_ = settings.SetJson(keys.Influx, conf)
	}

	if reset {
		settings.Delete(keys.Hems)
	} else {
		log.DEBUG.Println("migrate hems")
		_ = settings.SetYaml(keys.Hems, conf)
	}

	if reset {
		settings.Delete(keys.EEBus)
	} else {
		log.DEBUG.Println("migrate eebus")
		_ = settings.SetYaml(keys.EEBus, conf)
	}

	if reset {
		settings.Delete(keys.ModbusProxy)
	} else {
		log.DEBUG.Println("migrate modbusproxy")
		_ = settings.SetYaml(keys.ModbusProxy, conf)
	}

	if reset {
		settings.Delete(keys.Messaging)
	} else {
		log.DEBUG.Println("migrate messaging")
		_ = settings.SetYaml(keys.Messaging, conf)
	}

	if reset {
		settings.Delete(keys.Tariffs)
	} else {
		log.DEBUG.Println("migrate tariffs")
		_ = settings.SetYaml(keys.Tariffs, conf)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
