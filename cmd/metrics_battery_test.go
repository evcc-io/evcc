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
		{Group: metrics.Battery, Title: "bat", Data: []metrics.Slot{
			{Energy: 1.0, ReturnEnergy: 0.4},
			{Energy: 2.0, ReturnEnergy: 1.6},
		}},
		// non-battery series must be ignored
		{Group: metrics.Grid, Title: "grid", Data: []metrics.Slot{
			{Energy: 5.0, ReturnEnergy: 3.0},
		}},
	}

	totals := metricsBatteryTotals(series)
	require.Len(t, totals, 1)
	require.InDelta(t, 3.0, totals["bat"].charge, 0.001)
	require.InDelta(t, 2.0, totals["bat"].discharge, 0.001)
}

func TestMetricsWriteBatteryTable(t *testing.T) {
	selected := []metrics.EntityInfo{
		{Group: metrics.Battery, Name: "db:1", Title: "Home"},
		{Group: metrics.Battery, Name: "db:2", Title: "Hyper2000"}, // removed device: stored db title remains
		{Group: metrics.Battery, Name: "db:3"},
	}
	totals := map[string]batteryTotals{
		"Home":      {charge: 10.0, discharge: 9.0},
		"Hyper2000": {charge: 4.0, discharge: 3.0},
		// db:3 deliberately absent: no data in the timeframe
	}

	var buf bytes.Buffer
	metricsWriteBatteryTable(&buf, selected, totals)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 4) // header + 3 rows

	require.Contains(t, lines[0], "efficiency")

	// db:1: stored title, efficiency = discharge/charge
	require.Contains(t, lines[1], "Home")
	require.Contains(t, lines[1], "10.000")
	require.Contains(t, lines[1], "9.000")
	require.Contains(t, lines[1], "90.0%")

	// db:2: removed device still joins via its stored title; label is that title
	require.Contains(t, lines[2], "Hyper2000")
	require.Contains(t, lines[2], "4.000")
	require.Contains(t, lines[2], "3.000")
	require.Contains(t, lines[2], "75.0%")

	// db:3: no title anywhere -> label falls back to the name; no data -> blank efficiency
	f := strings.Fields(lines[3])
	require.Equal(t, "db:3", f[0]) // name column
	require.Equal(t, "db:3", f[1]) // title column falls back to name
	require.Contains(t, lines[3], "0.000")
	require.NotContains(t, lines[3], "%")
}
