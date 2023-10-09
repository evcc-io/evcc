package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// tariffCmd represents the vehicle command
var tariffCmd = &cobra.Command{
	Use:   "tariff [name]",
	Short: "Query configured tariff",
	Args:  cobra.MaximumNArgs(1),
	Run:   runTariff,
}

func init() {
	rootCmd.AddCommand(tariffCmd)
}

func runTariff(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		fatal(err)
	}

	var name string
	if len(args) == 1 {
		name = args[0]
	}

	for key, cc := range map[string]config.Typed{
		"grid":    conf.Tariffs.Grid,
		"feedin":  conf.Tariffs.FeedIn,
		"co2":     conf.Tariffs.Co2,
		"planner": conf.Tariffs.Planner,
	} {
		if cc.Type == "" || (name != "" && key != name) {
			continue
		}

		if name == "" {
			fmt.Println(key + ":")
		}

		tf, err := tariff.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			fatal(err)
		}

		rates, err := tf.Rates()
		if err != nil {
			fatal(err)
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
		fmt.Fprintln(tw, "From\tTo\tPrice/Cost")
		const format = "2006-01-02 15:04:05"

		for _, r := range rates {
			fmt.Fprintf(tw, "%s\t%s\t%.3f\n", r.Start.Local().Format(format), r.End.Local().Format(format), r.Price)
		}
		tw.Flush()

		fmt.Println()
	}

	// wait for shutdown
	<-shutdownDoneC()
}
