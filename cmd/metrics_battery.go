package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsBatteryCmd represents the metrics battery command
var metricsBatteryCmd = &cobra.Command{
	Use:   "battery [name ...]",
	Short: "Compare battery charge and discharge energy",
	Long: `Compare charge and discharge energy per battery.

Without arguments all batteries are compared for the current day. Batteries can
be selected by name or title.`,
	Run: runMetricsBattery,
}

func init() {
	metricsCmd.AddCommand(metricsBatteryCmd)
	metricsBatteryCmd.Flags().String("range", "", "Quick timeframe: day, month or year")
	metricsBatteryCmd.Flags().String("from", "", "Start date as YYYY-MM-DD (default today)")
	metricsBatteryCmd.Flags().String("to", "", "End date as YYYY-MM-DD, inclusive (default today)")
	metricsBatteryCmd.MarkFlagsMutuallyExclusive("range", "from")
	metricsBatteryCmd.MarkFlagsMutuallyExclusive("range", "to")
}

func runMetricsBattery(cmd *cobra.Command, args []string) {
	setupMetrics(cmd)

	from, to, err := metricsTimeframe(cmd.Flag("range").Value.String(), cmd.Flag("from").Value.String(), cmd.Flag("to").Value.String())
	if err != nil {
		log.FATAL.Fatal(err)
	}

	entities, err := metrics.ListEntities()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// limit selectable entities to the battery group
	batteries := make([]metrics.EntityInfo, 0, len(entities))
	for _, e := range entities {
		if e.Group == metrics.Battery {
			batteries = append(batteries, e)
		}
	}
	if len(batteries) == 0 {
		log.FATAL.Fatal("no battery entities found")
	}

	title := metricsEntityTitle()

	selected, err := metricsSelectEntities(batteries, args, "", title)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	series, err := metrics.QueryEnergy(from, to, "month", false)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	metricsWriteBatteryTable(os.Stdout, selected, metricsBatteryTotals(series), title)
	fmt.Fprintln(os.Stderr, "\nvalues in kWh")
}

// batteryTotals holds the accumulated charge and discharge energy of a battery.
// For a battery entity charge is stored as import energy, discharge as export
// energy (see core/site.go updateBatteryMeters).
type batteryTotals struct {
	charge    float64
	discharge float64
}

// metricsBatteryTotals sums charge and discharge energy per battery entity.
func metricsBatteryTotals(series []metrics.Series) map[string]batteryTotals {
	res := make(map[string]batteryTotals)
	for _, s := range series {
		if s.Group != metrics.Battery {
			continue
		}
		t := res[s.Name]
		for _, slot := range s.Data {
			t.charge += slot.Energy
			t.discharge += slot.ReturnEnergy
		}
		res[s.Name] = t
	}
	return res
}

// metricsWriteBatteryTable renders one row per battery comparing charge and
// discharge energy. Efficiency is the discharge/charge ratio, left blank when
// no energy was charged.
func metricsWriteBatteryTable(w io.Writer, selected []metrics.EntityInfo, totals map[string]batteryTotals, title func(group, name string) string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "name\ttitle\tcharge\tdischarge\tefficiency")

	for _, e := range selected {
		t := totals[e.Name]

		efficiency := ""
		if t.charge > 0 {
			efficiency = fmt.Sprintf("%.1f%%", t.discharge/t.charge*100)
		}

		fmt.Fprintf(tw, "%s\t%s\t%.3f\t%.3f\t%s\n",
			e.Name, title(e.Group, e.Name), t.charge, t.discharge, efficiency)
	}

	tw.Flush()
}
