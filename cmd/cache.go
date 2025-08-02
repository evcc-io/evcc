package cmd

import (
	"github.com/spf13/cobra"
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cache entries",
}

func init() {
	rootCmd.AddCommand(cacheCmd)
}
