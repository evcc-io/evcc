package cmd

import (
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Inspect stored energy metrics",
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}

// setupMetrics loads the config file and opens the database. Both metrics
// subcommands need the metric tables and the device configuration.
func setupMetrics(cmd *cobra.Command) {
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}
}

// metricsGroupRank returns the canonical sort position of a metric group.
func metricsGroupRank(group string) int {
	if i := slices.Index(metrics.GroupOrder, group); i >= 0 {
		return i
	}
	return len(metrics.GroupOrder)
}

// metricsSortCanonical sorts entities by canonical group order, then name.
func metricsSortCanonical(entities []metrics.EntityInfo) {
	slices.SortFunc(entities, func(a, b metrics.EntityInfo) int {
		if d := metricsGroupRank(a.Group) - metricsGroupRank(b.Group); d != 0 {
			return d
		}
		return strings.Compare(a.Name, b.Name)
	})
}

// metricsFormatDate formats a date for display, empty for the zero time.
func metricsFormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02")
}

// metricsEntityLabel returns the display label for an entity: the title stored
// in the entities table (the single source of truth), or the name as fallback.
func metricsEntityLabel(e metrics.EntityInfo) string {
	if e.Title != "" {
		return e.Title
	}
	return e.Name
}
