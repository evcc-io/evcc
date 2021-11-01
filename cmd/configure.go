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
	configureCmd.Flags().String("lang", "", "Define the localization to be used (en, de)")
}

func runConfigure(cmd *cobra.Command, args []string) {
	impl := &configure.CmdConfigure{}

	lang := ""
	langFlag, err := cmd.Flags().GetString("lang")
	if err == nil {
		lang = langFlag
	}

	logLevel := ""
	logLevelFlag, err := cmd.Flags().GetString("log")
	if err == nil {
		logLevel = logLevelFlag
	}

	impl.Run(log, logLevel, lang)
}
