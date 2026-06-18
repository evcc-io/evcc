package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/stretchr/testify/require"
)

func TestMetricsBatteryTotals(t *testing.T) {
	series := []metrics.Series{
		{Group: metrics.Battery, Name: "db:1", Title: "bat", Data: []metrics.Slot{
			{Energy: 1.0, ReturnEnergy: 0.4},
			{Energy: 2.0, ReturnEnergy: 1.6},
		}},
		// non-battery series must be ignored
		{Group: metrics.Grid, Name: "db:2", Title: "grid", Data: []metrics.Slot{
			{Energy: 5.0, ReturnEnergy: 3.0},
		}},
	}

	totals := metricsBatteryTotals(series)
	require.Len(t, totals, 1)
	require.InDelta(t, 3.0, totals["db:1"].charge, 0.001)
	require.InDelta(t, 2.0, totals["db:1"].discharge, 0.001)
}

func TestMetricsWriteBatteryTable(t *testing.T) {
	selected := []metrics.EntityInfo{
		{Group: metrics.Battery, Name: "db:1"},
		{Group: metrics.Battery, Name: "db:2", Title: "Hyper2000"}, // removed device: only the db title remains
		{Group: metrics.Battery, Name: "db:3"},
	}
	totals := map[string]batteryTotals{
		"db:1": {charge: 10.0, discharge: 9.0},
		"db:2": {charge: 4.0, discharge: 3.0},
		// db:3 deliberately absent: no data in the timeframe
	}
	title := func(group, name string) string {
		if name == "db:1" {
			return "Home"
		}
		return ""
	}

	var buf bytes.Buffer
	metricsWriteBatteryTable(&buf, selected, totals, title)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 4) // header + 3 rows

	require.Contains(t, lines[0], "efficiency")

	// db:1: live config title, efficiency = discharge/charge
	require.Contains(t, lines[1], "Home")
	require.Contains(t, lines[1], "10.000")
	require.Contains(t, lines[1], "9.000")
	require.Contains(t, lines[1], "90.0%")

	// db:2: removed device still joins by name; label falls back to the db title
	require.Contains(t, lines[2], "Hyper2000")
	require.Contains(t, lines[2], "4.000")
	require.Contains(t, lines[2], "3.000")
	require.Contains(t, lines[2], "75.0%")

	// db:3: no data -> zero totals, blank efficiency
	require.Contains(t, lines[3], "db:3")
	require.Contains(t, lines[3], "0.000")
	require.NotContains(t, lines[3], "%")
}
