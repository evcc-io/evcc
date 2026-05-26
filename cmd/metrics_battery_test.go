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
		{Group: metrics.Battery, Name: "bat1"},
		{Group: metrics.Battery, Name: "bat2"},
	}
	totals := map[string]batteryTotals{
		"Home": {charge: 10.0, discharge: 9.0},
		// bat2 deliberately absent: no data in the timeframe
	}
	title := func(group, name string) string {
		if name == "bat1" {
			return "Home"
		}
		return ""
	}

	var buf bytes.Buffer
	metricsWriteBatteryTable(&buf, selected, totals, title)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 3) // header + 2 rows

	require.Contains(t, lines[0], "efficiency")

	// bat1: title resolved, efficiency = discharge/charge
	require.Contains(t, lines[1], "Home")
	require.Contains(t, lines[1], "10.000")
	require.Contains(t, lines[1], "9.000")
	require.Contains(t, lines[1], "90.0%")

	// bat2: no data -> zero totals, blank efficiency
	require.Contains(t, lines[2], "bat2")
	require.Contains(t, lines[2], "0.000")
	require.NotContains(t, lines[2], "%")
}
