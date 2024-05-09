package cmd

import (
	_ "embed"
	"fmt"
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
	err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed)

	if err != nil {
		log.FATAL.Println("config invalid:", err)
		os.Exit(1)
	} else {
		fmt.Println("config valid")
	}
}
