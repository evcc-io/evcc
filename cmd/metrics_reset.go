package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsResetCmd represents the metrics reset command
var metricsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the daily rollup after the stored slots were changed by SQL",
	Long:  "Drops the daily rollup, the slots are kept. A running evcc rebuilds the rollup within an hour, otherwise the next start does.",
	Args:  cobra.NoArgs,
	Run:   runMetricsReset,
}

func init() {
	metricsCmd.AddCommand(metricsResetCmd)
}

func runMetricsReset(cmd *cobra.Command, args []string) {
	setupMetrics(cmd)

	if err := metrics.ResetRollup(); err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println("daily rollup reset")
}
