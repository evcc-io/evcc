package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/spf13/cobra"
)

// settingsSetCmd represents the configure command
var settingsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration setting",
	Run:   runSettingsSet,
	Args:  cobra.ExactArgs(2),
}

func init() {
	settingsCmd.AddCommand(settingsSetCmd)
	settingsSetCmd.Flags().BoolP(flagForce, "f", false, "Force (no confirmation)")
}

func runSettingsSet(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	confirmation, _ := cmd.Flags().GetBool(flagForce)
	if !confirmation {
		msg := fmt.Sprintf("Set %s", args[0])
		if val, _ := settings.String(args[0]); val != "" {
			msg = fmt.Sprintf("Override %s (current value: %s)", args[0], val)
		}

		prompt := &survey.Confirm{
			Message: msg,
		}

		if err := survey.AskOne(prompt, &confirmation); err != nil {
			log.FATAL.Fatal(err)
		}
	}

	if confirmation {
		settings.SetString(args[0], args[1])
	}

	// wait for shutdown
	<-shutdownDoneC()
}
