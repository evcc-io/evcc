package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// passwordResetCmd represents the vehicle command
var passwordResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset password",
	Args:  cobra.ExactArgs(0),
	Run:   runPasswordReset,
}

func init() {
	passwordCmd.AddCommand(passwordResetCmd)
}

func runPasswordReset(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	prompt := &survey.Confirm{
		Message: "Are you sure?",
		Help:    "help",
	}

	var confirm bool
	if err := survey.AskOne(prompt, &confirm); err != nil {
		log.FATAL.Fatal(err)
	}

	log.INFO.Println(confirm)

	// wait for shutdown
	<-shutdownDoneC()
}
