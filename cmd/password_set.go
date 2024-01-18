package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// passwordSetCmd represents the vehicle command
var passwordSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set password",
	Args:  cobra.ExactArgs(0),
	Run:   runPasswordSet,
}

func init() {
	passwordCmd.AddCommand(passwordSetCmd)
	// passwordSetCmd.Flags().BoolP(flagReset, "r", false, flagResetDescription)
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

	log.INFO.Println(password)

	// wait for shutdown
	<-shutdownDoneC()
}
