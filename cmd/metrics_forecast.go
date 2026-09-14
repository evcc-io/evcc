package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsForecastCmd represents the metrics forecast command
var metricsForecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Compare solar forecast against actual PV production",
	Args:  cobra.NoArgs,
	Run:   runMetricsForecast,
}

func init() {
	metricsCmd.AddCommand(metricsForecastCmd)
	metricsForecastCmd.Flags().String("range", "", "Quick timeframe: day, month or year")
	metricsForecastCmd.Flags().String("from", "", "Start date as YYYY-MM-DD (default today)")
	metricsForecastCmd.Flags().String("to", "", "End date as YYYY-MM-DD, inclusive (default today)")
	metricsForecastCmd.MarkFlagsMutuallyExclusive("range", "from")
	metricsForecastCmd.MarkFlagsMutuallyExclusive("range", "to")
}

func runMetricsForecast(cmd *cobra.Command, args []string) {
	setupMetrics(cmd)

	from, to, err := metricsTimeframe(cmd.Flag("range").Value.String(), cmd.Flag("from").Value.String(), cmd.Flag("to").Value.String())
	if err != nil {
		log.FATAL.Fatal(err)
	}

	series, err := metrics.QueryEnergy(from, to, "month", false)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	forecast, actual := metricsForecastTotals(series)

	metricsWriteForecastTable(os.Stdout, forecast, actual)
	fmt.Fprintln(os.Stderr, "\nvalues in kWh")
}

// metricsForecastTotals sums forecasted solar energy and actual PV production
// over the given series.
func metricsForecastTotals(series []metrics.Series) (forecast, actual float64) {
	for _, s := range series {
		var sum float64
		for _, slot := range s.Data {
			sum += slot.Energy
		}

		switch s.Group {
		case metrics.Forecast:
			forecast += sum
		case metrics.PV:
			actual += sum
		}
	}
	return forecast, actual
}

// metricsWriteForecastTable renders a one-row comparison of forecasted versus
// actual solar energy. Accuracy is the actual/forecast ratio, left blank when
// nothing was forecast.
func metricsWriteForecastTable(w io.Writer, forecast, actual float64) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "forecast\tactual\taccuracy")

	accuracy := ""
	if forecast > 0 {
		accuracy = fmt.Sprintf("%.1f%%", actual/forecast*100)
	}
	fmt.Fprintf(tw, "%.3f\t%.3f\t%s\n", forecast, actual, accuracy)

	tw.Flush()
}
