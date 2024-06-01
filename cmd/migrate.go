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
}

func runMigrate(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// TODO remove yaml file
	settings.SetInt(keys.Interval, int64(conf.Interval))
	settings.SetString(keys.SponsorToken, conf.SponsorToken)

	err := settings.SetJson(keys.Mqtt, conf)

	if err == nil {
		err = settings.SetJson(keys.Network, conf)
	}
	if err == nil {
		err = settings.SetJson(keys.Influx, conf)
	}
	if err == nil {
		err = settings.SetYaml(keys.Hems, conf)
	}
	if err == nil {
		err = settings.SetYaml(keys.EEBus, conf)
	}
	if err == nil {
		err = settings.SetYaml(keys.ModbusProxy, conf)
	}
	if err == nil {
		err = settings.SetYaml(keys.Messaging, conf)
	}
	if err == nil {
		err = settings.SetYaml(keys.Tariffs, conf)
	}

	if err != nil {
		log.FATAL.Fatal(err)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
