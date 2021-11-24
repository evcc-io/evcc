package cmd

import (
	_ "embed"

	"github.com/evcc-io/evcc/cmd/configure"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	var lang string
	langFlag, err := cmd.Flags().GetString("lang")
	if err == nil {
		lang = langFlag
	}

	util.LogLevel(viper.GetString("log"), nil)

	impl.Run(log, lang)
}
