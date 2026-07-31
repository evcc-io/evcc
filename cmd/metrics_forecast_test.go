package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/stretchr/testify/require"
)

func TestMetricsForecastTotals(t *testing.T) {
	series := []metrics.Series{
		{Group: metrics.Forecast, Title: "forecast", Data: []metrics.Slot{
			{Energy: 10.0}, {Energy: 5.0},
		}},
		// actual PV production is summed across all pv entities
		{Group: metrics.PV, Title: "pv1", Data: []metrics.Slot{
			{Energy: 4.0}, {Energy: 3.0},
		}},
		{Group: metrics.PV, Title: "pv2", Data: []metrics.Slot{
			{Energy: 2.0},
		}},
		// other groups are ignored
		{Group: metrics.Grid, Title: "grid", Data: []metrics.Slot{
			{Energy: 99.0},
		}},
	}

	forecast, actual := metricsForecastTotals(series)
	require.InDelta(t, 15.0, forecast, 0.001)
	require.InDelta(t, 9.0, actual, 0.001)
}

func TestMetricsWriteForecastTable(t *testing.T) {
	var buf bytes.Buffer
	metricsWriteForecastTable(&buf, 100.0, 90.0)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 2) // header + 1 row
	require.Contains(t, lines[0], "accuracy")
	require.Contains(t, lines[1], "100.000")
	require.Contains(t, lines[1], "90.000")
	require.Contains(t, lines[1], "90.0%")

	// zero forecast -> blank accuracy
	buf.Reset()
	metricsWriteForecastTable(&buf, 0, 0)
	lines = strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.NotContains(t, lines[1], "%")
}
