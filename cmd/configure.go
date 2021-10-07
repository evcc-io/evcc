package cmd

import (
	_ "embed"

	"github.com/evcc-io/evcc/cmd/configure"
	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Create an EVCC configuration",
	Run:   runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) {
	impl := &configure.CmdConfigure{}
	impl.Run(log)
}
