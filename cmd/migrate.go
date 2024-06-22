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
	migrateCmd.Flags().BoolP(flagReset, "r", false, flagResetDescription)
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
		log.WARN.Println("resetting:")
	} else {
		log.WARN.Println("migrating:")
	}

	log.DEBUG.Println("- global settings")
	if reset {
		settings.Delete(keys.Interval)
		settings.Delete(keys.SponsorToken)
	} else {
		settings.SetInt(keys.Interval, int64(conf.Interval))
		settings.SetString(keys.SponsorToken, conf.SponsorToken)
	}

	log.DEBUG.Println("- network")
	if reset {
		settings.Delete(keys.Network)
	} else {
		_ = settings.SetJson(keys.Network, conf.Network)
	}

	log.DEBUG.Println("- mqtt")
	if reset {
		settings.Delete(keys.Mqtt)
	} else {
		_ = settings.SetJson(keys.Mqtt, conf.Mqtt)
	}

	log.DEBUG.Println("- influx")
	if reset {
		settings.Delete(keys.Influx)
	} else {
		_ = settings.SetJson(keys.Influx, conf.Influx)
	}

	log.DEBUG.Println("- hems")
	if reset {
		settings.Delete(keys.Hems)
	} else if conf.HEMS.Type != "" {
		_ = settings.SetYaml(keys.Hems, conf.HEMS)
	}

	log.DEBUG.Println("- eebus")
	if reset {
		settings.Delete(keys.EEBus)
	} else if conf.EEBus.URI != "" {
		_ = settings.SetYaml(keys.EEBus, conf.EEBus)
	}

	log.DEBUG.Println("- modbusproxy")
	if reset {
		settings.Delete(keys.ModbusProxy)
	} else if len(conf.ModbusProxy) > 0 {
		_ = settings.SetYaml(keys.ModbusProxy, conf.ModbusProxy)
	}

	log.DEBUG.Println("- messaging")
	if reset {
		settings.Delete(keys.Messaging)
	} else if len(conf.Messaging.Services) > 0 {
		_ = settings.SetYaml(keys.Messaging, conf.Messaging)
	}

	log.DEBUG.Println("- tariffs")
	if reset {
		settings.Delete(keys.Tariffs)
	} else if conf.Tariffs.Grid.Type != "" || conf.Tariffs.FeedIn.Type != "" || conf.Tariffs.Co2.Type != "" || conf.Tariffs.Planner.Type != "" {
		_ = settings.SetYaml(keys.Tariffs, conf.Tariffs)
	}

	log.DEBUG.Println("- circuits")
	if reset {
		settings.Delete(keys.Circuits)
	} else if len(conf.Circuits) > 0 {
		_ = settings.SetYaml(keys.Circuits, conf.Circuits)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
