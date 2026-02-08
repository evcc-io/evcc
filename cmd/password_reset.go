package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/spf13/cobra"
)

var passwordResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset password",
	Args:  cobra.ExactArgs(0),
	Run:   runPasswordReset,
}

func init() {
	passwordCmd.AddCommand(passwordResetCmd)
	passwordResetCmd.Flags().BoolP(flagForce, "f", false, "Force (no confirmation)")
}

func runPasswordReset(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	confirm, _ := cmd.Flags().GetBool(flagForce)

	if !confirm {
		prompt := &survey.Confirm{
			Message: "Are you sure?",
			Help:    "help",
		}

		if err := survey.AskOne(prompt, &confirm); err != nil {
			log.FATAL.Fatal(err)
		}
	}

	if confirm {
		auth.New().RemoveAdminPassword()
	}
}
