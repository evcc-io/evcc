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
	Short: "Migrate yaml to database (deprecated), reset only",
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

	if !reset {
		log.FATAL.Fatal("migrating is not supported anymore. Use the web UI to create or `evcc migrate --reset` to reset the configuration.")
	}

	log.INFO.Println("resetting:")
	log.INFO.Println("- global settings")
	settings.Delete(keys.Interval)
	settings.Delete(keys.SponsorToken)
	settings.Delete(keys.Title)

	log.INFO.Println("- network")
	settings.Delete(keys.Network)

	log.INFO.Println("- mqtt")
	settings.Delete(keys.Mqtt)

	log.INFO.Println("- influx")
	settings.Delete(keys.Influx)

	log.INFO.Println("- hems")
	settings.Delete(keys.Hems)

	log.INFO.Println("- eebus")
	settings.Delete(keys.EEBus)

	log.INFO.Println("- modbusproxy")
	settings.Delete(keys.ModbusProxy)

	log.INFO.Println("- messaging")
	settings.Delete(keys.Messaging)

	log.INFO.Println("- tariffs")
	settings.Delete(keys.Tariffs)

	log.INFO.Println("- circuits")
	settings.Delete(keys.Circuits)

	log.INFO.Println("- device configs")
	settings.Delete(keys.GridMeter)
	settings.Delete(keys.PvMeters)
	settings.Delete(keys.AuxMeters)
	settings.Delete(keys.ExtMeters)
	settings.Delete(keys.BatteryMeters)
	// clear config table
	result := db.Instance.Delete(&config.Config{}, "true")
	log.INFO.Printf("  %d entries deleted", result.RowsAffected)
	// wait for shutdown
	<-shutdownDoneC()
}
