package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsEntitiesCmd represents the metrics entities command
var metricsEntitiesCmd = &cobra.Command{
	Use:   "entities",
	Short: "List metric entities",
	Args:  cobra.NoArgs,
	Run:   runMetricsEntities,
}

func init() {
	metricsCmd.AddCommand(metricsEntitiesCmd)
}

func runMetricsEntities(cmd *cobra.Command, args []string) {
	setupMetrics(cmd)

	entities, err := metrics.ListEntities()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	metricsSortCanonical(entities)
	title := metricsEntityTitle()

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "group\tname\ttitle\tslots\tfirst\tlast")

	for _, e := range entities {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			e.Group, e.Name, title(e.Group, e.Name), e.Slots,
			metricsFormatDate(e.First), metricsFormatDate(e.Last))
	}

	tw.Flush()
}
