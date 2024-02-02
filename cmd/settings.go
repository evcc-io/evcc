package cmd

import (
	"github.com/spf13/cobra"
)

// settingsCmd represents the configure command
var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage configuration settings",
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
