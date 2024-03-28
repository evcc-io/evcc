package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/spf13/cobra"
)

var passwordSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set password",
	Args:  cobra.ExactArgs(0),
	Run:   runPasswordSet,
}

func init() {
	passwordCmd.AddCommand(passwordSetCmd)
}

func runPasswordSet(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	prompt := &survey.Password{
		Message: "Password",
		Help:    "help",
	}

	var password string
	if err := survey.AskOne(prompt, &password); err != nil {
		log.FATAL.Fatal(err)
	}

	if password == "" {
		log.FATAL.Fatal("password cannot be empty")
	} else {
		auth.New().SetAdminPassword(password)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
