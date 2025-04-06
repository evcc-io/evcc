package cmd

import (
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
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

	reset := cmd.Flag(flagReset).Changed

	// TODO remove yaml file
	if reset {
		log.INFO.Println("resetting:")
	} else {
		log.INFO.Println("migrating:")
	}

	log.INFO.Println("- global settings")
	if reset {
		settings.Delete(keys.Interval)
		settings.Delete(keys.SponsorToken)
		settings.Delete(keys.Title)
	} else {
		settings.SetInt(keys.Interval, int64(conf.Interval))
		settings.SetString(keys.SponsorToken, conf.SponsorToken)
	}

	log.INFO.Println("- network")
	if reset {
		settings.Delete(keys.Network)
	} else {
		_ = settings.SetJson(keys.Network, conf.Network)
	}

	log.INFO.Println("- mqtt")
	if reset {
		settings.Delete(keys.Mqtt)
	} else {
		_ = settings.SetJson(keys.Mqtt, conf.Mqtt)
	}

	log.INFO.Println("- influx")
	if reset {
		settings.Delete(keys.Influx)
	} else {
		_ = settings.SetJson(keys.Influx, conf.Influx)
	}

	log.INFO.Println("- hems")
	if reset {
		settings.Delete(keys.Hems)
	} else if conf.HEMS.Type != "" {
		_ = settings.SetYaml(keys.Hems, conf.HEMS)
	}

	log.INFO.Println("- eebus")
	if reset {
		settings.Delete(keys.EEBus)
	} else if conf.EEBus.Configured() {
		_ = settings.SetYaml(keys.EEBus, conf.EEBus)
	}

	log.INFO.Println("- modbusproxy")
	if reset {
		settings.Delete(keys.ModbusProxy)
	} else if len(conf.ModbusProxy) > 0 {
		_ = settings.SetYaml(keys.ModbusProxy, conf.ModbusProxy)
	}

	log.INFO.Println("- messaging")
	if reset {
		settings.Delete(keys.Messaging)
	} else if len(conf.Messaging.Services) > 0 {
		_ = settings.SetYaml(keys.Messaging, conf.Messaging)
	}

	log.INFO.Println("- tariffs")
	if reset {
		settings.Delete(keys.Tariffs)
	} else if conf.Tariffs.Grid.Type != "" || conf.Tariffs.FeedIn.Type != "" || conf.Tariffs.Co2.Type != "" || conf.Tariffs.Planner.Type != "" {
		_ = settings.SetYaml(keys.Tariffs, conf.Tariffs)
	}

	log.INFO.Println("- circuits")
	if reset {
		settings.Delete(keys.Circuits)
	} else if len(conf.Circuits) > 0 {
		_ = settings.SetYaml(keys.Circuits, conf.Circuits)
	}

	log.INFO.Println("- device configs")
	if reset {
		// site keys
		settings.Delete(keys.GridMeter)
		settings.Delete(keys.PvMeters)
		settings.Delete(keys.AuxMeters)
		settings.Delete(keys.BatteryMeters)
		// clear config table
		result := db.Instance.Delete(&config.Config{}, "true")
		log.INFO.Printf("  %d entries deleted", result.RowsAffected)
	}
	// wait for shutdown
	<-shutdownDoneC()
}
