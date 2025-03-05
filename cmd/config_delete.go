package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cobra"
)

// configDeleteCmd represents the configure command
var configDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete device",
	Run:   runConfigDelete,
	Args:  cobra.ExactArgs(1),
}

func init() {
	configCmd.AddCommand(configDeleteCmd)
}

func runConfigDelete(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	id, err := config.IDForName(args[0])
	if err != nil {
		log.FATAL.Fatal(err)
	}

	c, err := config.ConfigByID(id)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println(config.NameForID(c.ID), "type:"+c.Type, c.Value)

	switch c.Class {
	case templates.Charger:
		deleteDevice[api.Charger](c)
	case templates.Meter:
		deleteDevice[api.Meter](c)
	case templates.Vehicle:
		deleteDevice[api.Vehicle](c)
	case templates.Tariff:
		deleteDevice[api.Tariff](c)
	case templates.Circuit:
		deleteDevice[api.Circuit](c)
	case templates.Loadpoint:
		deleteDevice[loadpoint.API](c)
	}
}

func deleteDevice[T any](c config.Config) {
	var zero T
	dev := config.NewConfigurableDevice[T](&c, zero)
	if err := dev.Delete(); err != nil {
		log.FATAL.Fatal(err)
	}
}
