package metrics

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/locale"
	"github.com/stretchr/testify/require"
)

func mkSlot(t time.Time, energy, returnEnergy float64) Slot {
	return Slot{Start: t, End: t.Add(15 * time.Minute), Energy: energy, ReturnEnergy: returnEnergy}
}

func TestSeriesCSV_HeaderAndLayout(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)

	series := SeriesCSV{
		{Group: PV, Name: "pv", Data: []Slot{mkSlot(t0, 2.5, 0), mkSlot(t1, 3.0, 0)}},
		{Group: Battery, Name: "battery-home", Data: []Slot{mkSlot(t0, 0, 0.7800), mkSlot(t1, 0.5300, 0)}},
		{Group: Battery, Name: "battery-garage", Data: []Slot{mkSlot(t0, 0, 1.2500), mkSlot(t1, 0, 0)}},
		{Group: Grid, Name: "grid", Data: []Slot{mkSlot(t0, 0, 0.4123), mkSlot(t1, 0.1, 0)}},
		{Group: Home, Name: "home", Data: []Slot{mkSlot(t0, 0.3661, 0), mkSlot(t1, 0.4, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, series.WriteCsv(context.Background(), &buf))

	// UTF-8 BOM
	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}), "expected UTF-8 BOM")

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF})))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3, "expected header + 2 data rows")

	header := rows[0]
	expected := []string{
		"time.start", "time.end",
		"pv.pv.energy.Wh",
		"battery.battery-garage.energy.Wh", "battery.battery-garage.returnEnergy.Wh",
		"battery.battery-home.energy.Wh", "battery.battery-home.returnEnergy.Wh",
		"grid.grid.energy.Wh", "grid.grid.returnEnergy.Wh",
		"home.home.energy.Wh",
	}
	require.Equal(t, expected, header, "header order: GROUP_ORDER, alphabetical entities; returnEnergy only for grid/battery")

	// time.end = time.start + slot length (15 min in mkSlot)
	require.Equal(t, t0.Local().Format("2006-01-02 15:04:05"), rows[1][0])
	require.Equal(t, t0.Add(15*time.Minute).Local().Format("2006-01-02 15:04:05"), rows[1][1])

	// First data row values (t0) — plain Wh integers. Cols start at index 2.
	require.Equal(t, "2500", rows[1][2]) // pv.pv energy
	require.Equal(t, "0", rows[1][3])    // battery-garage energy
	require.Equal(t, "1250", rows[1][4]) // battery-garage returnEnergy
	require.Equal(t, "0", rows[1][5])    // battery-home energy
	require.Equal(t, "780", rows[1][6])  // battery-home returnEnergy
	require.Equal(t, "0", rows[1][7])    // grid energy
	require.Equal(t, "412", rows[1][8])  // grid returnEnergy
	require.Equal(t, "366", rows[1][9])  // home energy
}

func TestSeriesCSV_GermanLocale(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	series := SeriesCSV{
		{Group: PV, Name: "pv", Data: []Slot{mkSlot(t0, 2.5, 0)}},
	}

	var buf bytes.Buffer
	ctx := context.WithValue(context.Background(), locale.Locale, "de")
	require.NoError(t, series.WriteCsv(ctx, &buf))

	s := string(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}))
	require.True(t, strings.Contains(s, ";"), "german CSV must use ';' separator")
	// Values are plain Wh integers so they're locale-agnostic. We just check
	// no decimal point / comma snuck in.
	require.True(t, strings.Contains(s, "2500"), "value should be Wh integer")
	require.False(t, strings.Contains(s, "2.500"), "no '.' decimal/thousands")
	require.False(t, strings.Contains(s, "2,500"), "no ',' decimal/thousands")
}

func TestSeriesCSV_MissingSlotIsEmpty(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)

	// Two entities, second one only has data for the first timestamp → second
	// timestamp must produce an empty cell rather than 0.000.
	series := SeriesCSV{
		{Group: PV, Name: "a", Data: []Slot{mkSlot(t0, 1, 0), mkSlot(t1, 2, 0)}},
		{Group: PV, Name: "b", Data: []Slot{mkSlot(t0, 3, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, series.WriteCsv(context.Background(), &buf))

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF})))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)

	require.Equal(t, []string{"time.start", "time.end", "pv.a.energy.Wh", "pv.b.energy.Wh"}, rows[0])

	// row 1: t0 has both entities (Wh integers); cols start at index 2.
	require.Equal(t, "1000", rows[1][2])
	require.Equal(t, "3000", rows[1][3])

	// row 2: t1 has only entity a
	require.Equal(t, "2000", rows[2][2])
	require.Equal(t, "", rows[2][3], "missing slot must be empty, not zero")
}

func TestSeriesCSV_BatteryHasReturnEnergyColumn(t *testing.T) {
	// Battery is bidirectional, so even a series with zero discharge keeps the
	// returnEnergy column.
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	series := SeriesCSV{
		{Group: Battery, Name: "bat", Data: []Slot{mkSlot(t0, 0.5, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, series.WriteCsv(context.Background(), &buf))

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF})))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{"time.start", "time.end", "battery.bat.energy.Wh", "battery.bat.returnEnergy.Wh"}, rows[0])
	require.Equal(t, "500", rows[1][2])
	require.Equal(t, "0", rows[1][3])
}
