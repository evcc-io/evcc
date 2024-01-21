package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"
)

var checkconfig = &cobra.Command{
	Use:   "checkconfig",
	Short: "Check config file for errors",
	Run:   runConfigCheck,
}

func init() {
	rootCmd.AddCommand(checkconfig)
}

func runConfigCheck(cmd *cobra.Command, args []string) {
	cfgErr := loadConfigFile(&conf)

	if cfgErr != nil {
		log.FATAL.Println("Config not valid")
		log.FATAL.Println(cfgErr)
		os.Exit(1)
	} else {
		log.INFO.Println("Config is valid")
	}
}
