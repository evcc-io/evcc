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
	configureCmd.Flags().String("mode", configure.RenderingMode_Simple, "Define the rendering mode ("+configure.RenderingMode_Simple+", "+configure.RenderingMode_Advanced+")")
}

func runConfigure(cmd *cobra.Command, args []string) {
	impl := &configure.CmdConfigure{}

	lang, err := cmd.Flags().GetString("lang")
	if err != nil {
		panic(err)
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		panic(err)
	}

	util.LogLevel(viper.GetString("log"), nil)

	impl.Run(log, lang, mode)
}
