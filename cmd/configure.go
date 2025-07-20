package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/evcc-io/evcc/cmd/configure"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Create configuration (evcc.yaml)",
	Run:   runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().String("lang", "", "Define the localization to be used (en, de)")
	configureCmd.Flags().Bool("advanced", false, "Enables handling of advanced configuration options")
	configureCmd.Flags().Bool("expand", false, "Enables rendering expanded configuration files")
	configureCmd.Flags().String("category", "", "Pre-select device category for advanced configuration (implies advanced)")
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

	category, err := cmd.Flags().GetString("category")
	if err != nil {
		panic(err)
	}

	util.LogLevel(viper.GetString("log"), nil)

	// catch signals
	go func() {
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, os.Interrupt, syscall.SIGTERM)

		<-signalC // wait for signal

		os.Exit(1)
	}()

	impl.Run(log, lang, advanced, expand, category)
}
