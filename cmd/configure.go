package cmd

import (
	_ "embed"

	"github.com/evcc-io/evcc/cmd/configure"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/shutdown"
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
	configureCmd.Flags().Bool("advanced", false, "Enables handling of advanced configuration options")
	configureCmd.Flags().Bool("expand", false, "Enables rendering expanded configuration files")
}

func runConfigure(cmd *cobra.Command, args []string) {
	impl := &configure.CmdConfigure{}

	lang, err := cmd.Flags().GetString("lang")
	if err != nil {
		log.FATAL.Fatal(err)
	}

	advanced, err := cmd.Flags().GetBool("advanced")
	if err != nil {
		panic(err)
	}

	expand, err := cmd.Flags().GetBool("expand")
	if err != nil {
		panic(err)
	}

	util.LogLevel(viper.GetString("log"), nil)

	stopC := make(chan struct{})
	go shutdown.Run(stopC)

	impl.Run(log, lang, advanced, expand)

	close(stopC)
	<-shutdown.Done()
}
