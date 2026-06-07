package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/stretchr/testify/require"
)

func TestMetricsTimeframe(t *testing.T) {
	// explicit range: to is inclusive, so it extends to the end of the named day
	from, to, err := metricsTimeframe("", "2026-05-01", "2026-05-03")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local), from)
	require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local), to)

	// to before from
	_, _, err = metricsTimeframe("", "2026-05-05", "2026-05-01")
	require.Error(t, err)

	// invalid date
	_, _, err = metricsTimeframe("", "notadate", "")
	require.Error(t, err)
}

func TestMetricsTimeframeRange(t *testing.T) {
	now := time.Now()

	// day
	from, to, err := metricsTimeframe("day", "", "")
	require.NoError(t, err)
	require.Equal(t, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local), from)
	require.Equal(t, from.AddDate(0, 0, 1), to)

	// month, case-insensitive
	from, to, err = metricsTimeframe("Month", "", "")
	require.NoError(t, err)
	require.Equal(t, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), from)
	require.Equal(t, from.AddDate(0, 1, 0), to)

	// year
	from, to, err = metricsTimeframe("year", "", "")
	require.NoError(t, err)
	require.Equal(t, time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local), from)
	require.Equal(t, from.AddDate(1, 0, 0), to)

	// invalid range value
	_, _, err = metricsTimeframe("decade", "", "")
	require.Error(t, err)
}

func TestMetricsSelectEntities(t *testing.T) {
	entities := []metrics.EntityInfo{
		{Group: metrics.Grid, Name: "grid"},
		{Group: metrics.PV, Name: "pv1"},
		{Group: metrics.Loadpoint, Name: "lp-1"},
	}
	title := func(group, name string) string {
		if group == metrics.Loadpoint && name == "lp-1" {
			return "Carport"
		}
		return ""
	}

	// explicit selectors match by name or title and preserve argument order
	res, err := metricsSelectEntities(entities, []string{"Carport", "grid"}, "", title)
	require.NoError(t, err)
	require.Equal(t, []string{"lp-1", "grid"}, []string{res[0].Name, res[1].Name})

	// unknown selector errors
	_, err = metricsSelectEntities(entities, []string{"bogus"}, "", title)
	require.Error(t, err)

	// no selectors: all entities in canonical group order (pv before grid)
	res, err = metricsSelectEntities(entities, nil, "", title)
	require.NoError(t, err)
	require.Equal(t, metrics.PV, res[0].Group)

	// empty group errors
	_, err = metricsSelectEntities(entities, nil, metrics.Battery, title)
	require.Error(t, err)
}

func TestMetricsWriteTable(t *testing.T) {
	h0 := time.Date(2026, 5, 22, 0, 0, 0, 0, time.Local)
	h1 := h0.Add(time.Hour)

	selected := []metrics.EntityInfo{
		{Group: metrics.Loadpoint, Name: "lp-1"},
		{Group: metrics.Grid, Name: "grid"},
	}
	byEntity := map[string]metrics.Series{
		metrics.Loadpoint + "/Carport": {Group: metrics.Loadpoint, Title: "Carport", Data: []metrics.Slot{
			{Start: h0, Energy: 1.84},
		}},
		metrics.Grid + "/grid": {Group: metrics.Grid, Title: "grid", Data: []metrics.Slot{
			{Start: h0, Energy: 0.412},
			{Start: h1, Energy: 0.38, ReturnEnergy: 0.05},
		}},
	}
	title := func(group, name string) string {
		if group == metrics.Loadpoint {
			return "Carport"
		}
		if group == metrics.Grid {
			return "grid"
		}
		return ""
	}

	var buf bytes.Buffer
	metricsWriteTable(&buf, selected, byEntity, title, "hour")

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 3) // header + 2 rows

	// header: title falls back to name; bidirectional grid gets a return column
	require.Contains(t, lines[0], "Carport")
	require.Contains(t, lines[0], "grid↑")

	// row 1: both lp-1 and grid have data
	require.Contains(t, lines[1], "2026-05-22 00:00")
	require.Contains(t, lines[1], "1.840")
	require.Contains(t, lines[1], "0.412")

	// row 2: lp-1 has no slot -> blank cell, not 0.000; grid return energy shown
	require.Contains(t, lines[2], "2026-05-22 01:00")
	require.Contains(t, lines[2], "0.380")
	require.Contains(t, lines[2], "0.050")
	require.NotContains(t, lines[2], "0.000")
}
