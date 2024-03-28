package cmd

import (
	"github.com/spf13/cobra"
)

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Password administration",
}

func init() {
	rootCmd.AddCommand(passwordCmd)
}
