package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/spf13/cobra"
)

// metricsDataCmd represents the metrics data command
var metricsDataCmd = &cobra.Command{
	Use:   "data [entity ...]",
	Short: "Export energy data as a table",
	Long: `Export aggregated energy data as a table.

Without arguments all entities are exported for the current day. Entities can be
selected by name or title; run the entities subcommand to list them.`,
	Run: runMetricsData,
}

func init() {
	metricsCmd.AddCommand(metricsDataCmd)
	metricsDataCmd.Flags().String("range", "", "Quick timeframe: day, month or year")
	metricsDataCmd.Flags().String("from", "", "Start date as YYYY-MM-DD (default today)")
	metricsDataCmd.Flags().String("to", "", "End date as YYYY-MM-DD, inclusive (default today)")
	metricsDataCmd.Flags().String("aggregate", "hour", "Aggregation interval: 15m, hour, day or month")
	metricsDataCmd.Flags().String("group", "", "Limit output to an entity group")
	metricsDataCmd.Flags().Bool("csv", false, "Output CSV instead of a table")
	metricsDataCmd.MarkFlagsMutuallyExclusive("range", "from")
	metricsDataCmd.MarkFlagsMutuallyExclusive("range", "to")
}

func runMetricsData(cmd *cobra.Command, args []string) {
	setupMetrics(cmd)

	group := cmd.Flag("group").Value.String()
	if group != "" && len(args) > 0 {
		log.FATAL.Fatal("--group and entity arguments are mutually exclusive")
	}

	from, to, err := metricsTimeframe(cmd.Flag("range").Value.String(), cmd.Flag("from").Value.String(), cmd.Flag("to").Value.String())
	if err != nil {
		log.FATAL.Fatal(err)
	}

	aggregate := cmd.Flag("aggregate").Value.String()

	entities, err := metrics.ListEntities()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	selected, err := metricsSelectEntities(entities, args, group)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	series, err := metrics.QueryEnergy(from, to, aggregate, false)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	byEntity := make(map[string]metrics.Series, len(series))
	for _, s := range series {
		byEntity[s.Group+"/"+s.Title] = s
	}

	if asCSV, _ := cmd.Flags().GetBool("csv"); asCSV {
		var out metrics.SeriesCSV
		seen := make(map[string]bool, len(selected))
		for _, e := range selected {
			key := e.Group + "/" + metricsEntityLabel(e)
			if seen[key] {
				continue
			}
			seen[key] = true
			if s, ok := byEntity[key]; ok {
				out = append(out, s)
			}
		}
		if err := out.WriteCsv(context.Background(), os.Stdout); err != nil {
			log.FATAL.Fatal(err)
		}
		return
	}

	metricsWriteTable(os.Stdout, selected, byEntity, aggregate)
	fmt.Fprintln(os.Stderr, "\nvalues in kWh")
}

// metricsTimeframe resolves the from/to query bounds. A non-empty range string
// (today, month, year) takes precedence and is mutually exclusive with the
// from/to date flags. The to date is inclusive; an empty timeframe defaults to
// the current day.
func metricsTimeframe(rangeStr, fromStr, toStr string) (time.Time, time.Time, error) {
	const layout = "2006-01-02"

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	if rangeStr != "" {
		switch strings.ToLower(rangeStr) {
		case "day":
			return today, today.AddDate(0, 0, 1), nil
		case "month":
			from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
			return from, from.AddDate(0, 1, 0), nil
		case "year":
			from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)
			return from, from.AddDate(1, 0, 0), nil
		default:
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --range %q (day, month or year)", rangeStr)
		}
	}

	from := today
	if fromStr != "" {
		t, err := time.ParseInLocation(layout, fromStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from date: %w", err)
		}
		from = t
	}

	to := today.AddDate(0, 0, 1)
	if toStr != "" {
		t, err := time.ParseInLocation(layout, toStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to date: %w", err)
		}
		to = t.AddDate(0, 0, 1) // inclusive end day
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, errors.New("--to must not be before --from")
	}

	return from, to, nil
}

// metricsSelectEntities resolves the entities to export. Without selectors all
// entities (optionally limited to a group) are returned in canonical order;
// explicit selectors match by name or title and preserve the requested order.
func metricsSelectEntities(entities []metrics.EntityInfo, args []string, group string) ([]metrics.EntityInfo, error) {
	if len(args) == 0 {
		res := make([]metrics.EntityInfo, 0, len(entities))
		for _, e := range entities {
			if group == "" || e.Group == group {
				res = append(res, e)
			}
		}
		if group != "" && len(res) == 0 {
			return nil, fmt.Errorf("no entities in group %q", group)
		}
		metricsSortCanonical(res)
		return res, nil
	}

	var res []metrics.EntityInfo
	for _, arg := range args {
		var matched []metrics.EntityInfo
		for _, e := range entities {
			if e.Name == arg || e.Title == arg {
				matched = append(matched, e)
			}
		}
		if len(matched) == 0 {
			return nil, fmt.Errorf("unknown entity %q", arg)
		}
		res = append(res, matched...)
	}
	return res, nil
}

// metricsTimeLayout returns the time column format for the given aggregation.
func metricsTimeLayout(aggregate string) string {
	switch aggregate {
	case "day":
		return "2006-01-02"
	case "month":
		return "2006-01"
	default: // 15m, hour
		return "2006-01-02 15:04"
	}
}

// metricsWriteTable renders the wide energy table: one row per time slot, one
// column per entity, plus a second column for the export energy of
// bidirectional entities (grid, battery).
func metricsWriteTable(w io.Writer, selected []metrics.EntityInfo, byEntity map[string]metrics.Series, aggregate string) {
	layout := metricsTimeLayout(aggregate)

	type colSpec struct {
		entity    metrics.EntityInfo
		energyCol int
		returnCol int // -1 unless the entity is bidirectional
	}

	header := []string{"time"}
	var specs []colSpec

	for _, e := range selected {
		label := metricsEntityLabel(e)

		spec := colSpec{entity: e, energyCol: len(header) - 1, returnCol: -1}
		header = append(header, label)

		if e.Group == metrics.Grid || e.Group == metrics.Battery {
			spec.returnCol = len(header) - 1
			header = append(header, label+"↑")
		}

		specs = append(specs, spec)
	}

	ncols := len(header) - 1

	rowByKey := make(map[string][]string)
	var rowKeys []string

	cells := func(key string) []string {
		if c, ok := rowByKey[key]; ok {
			return c
		}
		c := make([]string, ncols)
		rowByKey[key] = c
		rowKeys = append(rowKeys, key)
		return c
	}

	for _, spec := range specs {
		s, ok := byEntity[spec.entity.Group+"/"+metricsEntityLabel(spec.entity)]
		if !ok {
			continue
		}
		for _, slot := range s.Data {
			row := cells(slot.Start.Local().Format(layout))
			row[spec.energyCol] = fmt.Sprintf("%.3f", slot.Energy)
			if spec.returnCol >= 0 {
				row[spec.returnCol] = fmt.Sprintf("%.3f", slot.ReturnEnergy)
			}
		}
	}

	slices.Sort(rowKeys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, key := range rowKeys {
		fmt.Fprintln(tw, key+"\t"+strings.Join(rowByKey[key], "\t"))
	}
	tw.Flush()
}
