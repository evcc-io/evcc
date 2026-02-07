package cmd

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/redact"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cobra"
)

// configCmd represents the configure command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Dump database configuration",
	Run:   runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().String("class", "", "Device class ("+strings.Join(templates.ClassStrings(), "|")+")")
}

func runConfig(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	cc := templates.ClassValues()
	if c := cmd.Flag("class").Value.String(); c != "" {
		class, err := templates.ClassString(c)
		if err != nil {
			log.FATAL.Fatal(err)
		}
		cc = []templates.Class{class}
	}

	for _, class := range cc {
		configurable, err := config.ConfigurationsByClass(class)
		if err != nil {
			log.FATAL.Fatal(err)
		}

		if len(configurable) > 0 {
			if len(cc) > 0 {
				fmt.Println(class)
				fmt.Println("---")
			}

			for _, c := range configurable {
				fmt.Println(config.NameForID(c.ID), fmt.Sprintf("%+v", c.Properties), redact.Map(c.Data))
			}

			fmt.Println("")
		}
	}
}
