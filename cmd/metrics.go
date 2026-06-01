package cmd

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
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

// metricsEntityTitle resolves human-readable titles for metric entities. Titles
// exist only for configured loadpoints and meters; virtual entities (home,
// forecast) have none. The returned function maps an entity to its title, or an
// empty string when no title is configured.
func metricsEntityTitle() func(group, name string) string {
	// loadpoints are addressed as lp-<n>, numbered yaml-first then database,
	// mirroring configureLoadpoints
	loadpoints := make(map[string]string)
	idx := 0
	addLoadpoint := func(n config.Named) {
		idx++
		if t, ok := n.Property("title").(string); ok && t != "" {
			loadpoints["lp-"+strconv.Itoa(idx)] = t
		}
	}
	for _, lp := range conf.Loadpoints {
		addLoadpoint(lp)
	}
	if devices, err := config.ConfigurationsByClass(templates.Loadpoint); err == nil {
		for _, dev := range devices {
			addLoadpoint(dev.Named())
		}
	}

	// meter entities are addressed by their device ref; the title comes from the
	// device configuration
	meters := make(map[string]string)
	for _, m := range conf.Meters {
		if t, ok := m.Property("title").(string); ok && t != "" {
			meters[m.Name] = t
		}
	}
	if devices, err := config.ConfigurationsByClass(templates.Meter); err == nil {
		for _, dev := range devices {
			if dev.Title != "" {
				meters[config.NameForID(dev.ID)] = dev.Title
			}
		}
	}

	return func(group, name string) string {
		switch group {
		case metrics.Loadpoint:
			return loadpoints[name]
		case metrics.Grid, metrics.PV, metrics.Battery, metrics.Meter:
			return meters[name]
		default:
			return ""
		}
	}
}
