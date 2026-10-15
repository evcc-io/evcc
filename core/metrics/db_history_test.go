package metrics

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	csvexport "github.com/evcc-io/evcc/util/export/csv"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/stretchr/testify/require"
)

func mkSlot(t time.Time, energy, returnEnergy float64) Slot {
	return Slot{Start: t, End: t.Add(15 * time.Minute), Energy: energy, ReturnEnergy: returnEnergy}
}

// writeSeriesCsv exports s to w as localized CSV via the csv row writer.
func writeSeriesCsv(t *testing.T, ctx context.Context, s SeriesExport, w io.Writer) error {
	ww, err := csvexport.New(ctx, w)
	require.NoError(t, err)
	return s.Write(ww)
}

func TestSeriesExport_HeaderAndLayout(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)

	series := SeriesExport{
		{Group: PV, Title: "pv", Data: []Slot{mkSlot(t0, 2.5, 0), mkSlot(t1, 3.0, 0)}},
		{Group: Battery, Title: "battery-home", Data: []Slot{mkSlot(t0, 0, 0.7800), mkSlot(t1, 0.5300, 0)}},
		{Group: Battery, Title: "battery-garage", Data: []Slot{mkSlot(t0, 0, 1.2500), mkSlot(t1, 0, 0)}},
		{Group: Grid, Title: "grid", Data: []Slot{mkSlot(t0, 0, 0.4123), mkSlot(t1, 0.1, 0)}},
		{Group: Home, Title: "home", Data: []Slot{mkSlot(t0, 0.3661, 0), mkSlot(t1, 0.4, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, writeSeriesCsv(t, context.Background(), series, &buf))

	// UTF-8 BOM
	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}), "expected UTF-8 BOM")

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF})))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3, "expected header + 2 data rows")

	header := rows[0]
	expected := []string{
		"time.start", "time.end",
		"pv.energy.Wh",
		"battery.battery-garage.energy.Wh", "battery.battery-garage.returnEnergy.Wh",
		"battery.battery-home.energy.Wh", "battery.battery-home.returnEnergy.Wh",
		"grid.energy.Wh", "grid.returnEnergy.Wh",
		"home.energy.Wh",
	}
	require.Equal(t, expected, header, "single-entity groups omit the title level; multi-entity groups include the title")

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

func TestSeriesExport_GermanLocale(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	series := SeriesExport{
		{Group: PV, Title: "pv", Data: []Slot{mkSlot(t0, 2.5, 0)}},
	}

	var buf bytes.Buffer
	ctx := context.WithValue(context.Background(), locale.Locale, "de")
	require.NoError(t, writeSeriesCsv(t, ctx, series, &buf))

	s := string(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}))
	require.False(t, strings.Contains(s, ";"), "delimiter is ',' regardless of locale")
	require.True(t, strings.Contains(s, "2500"), "value should be Wh integer")
	require.False(t, strings.Contains(s, "2.500"), "no '.' decimal/thousands")
	require.False(t, strings.Contains(s, "2,500"), "no ',' decimal/thousands")
}

func TestSeriesExport_MissingSlotIsEmpty(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)

	// Two entities, second one only has data for the first timestamp → second
	// timestamp must produce an empty cell rather than 0.000.
	series := SeriesExport{
		{Group: PV, Title: "a", Data: []Slot{mkSlot(t0, 1, 0), mkSlot(t1, 2, 0)}},
		{Group: PV, Title: "b", Data: []Slot{mkSlot(t0, 3, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, writeSeriesCsv(t, context.Background(), series, &buf))

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

func TestSeriesExport_BatteryHasReturnEnergyColumn(t *testing.T) {
	// Battery is bidirectional, so even a series with zero discharge keeps the
	// returnEnergy column.
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	series := SeriesExport{
		{Group: Battery, Title: "bat", Data: []Slot{mkSlot(t0, 0.5, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, writeSeriesCsv(t, context.Background(), series, &buf))

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF})))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{"time.start", "time.end", "battery.bat.energy.Wh", "battery.bat.returnEnergy.Wh"}, rows[0])
	require.Equal(t, "500", rows[1][2])
	require.Equal(t, "0", rows[1][3])
}

func TestSeriesExport_SocTempColumns(t *testing.T) {
	t0 := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)

	slot := func(t time.Time, socTemp *float64) Slot {
		s := mkSlot(t, 1, 0)
		s.SocTemp = socTemp
		return s
	}

	series := SeriesExport{
		{Group: Battery, Title: "bat", Data: []Slot{slot(t0, new(90.0)), slot(t1, new(80.0))}},
		{Group: Loadpoint, Title: "heater", IsTemp: true, Data: []Slot{slot(t0, new(45.0)), slot(t1, new(46.0))}},
		{Group: PV, Title: "pv", Data: []Slot{mkSlot(t0, 2, 0), mkSlot(t1, 3, 0)}},
	}

	var buf bytes.Buffer
	require.NoError(t, writeSeriesCsv(t, context.Background(), series, &buf))
	out := buf.String()

	require.Contains(t, out, "battery.bat.soc.pct")
	require.Contains(t, out, "loadpoint.heater.temp.degC")
	require.NotContains(t, out, "battery.bat.temp")
	require.NotContains(t, out, "pv.soc")
}

func TestQueryEnergySoc(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Id: 2, Name: "bat", Group: Battery}
	require.NoError(t, db.Instance.Create(&e).Error)

	base := time.Date(2026, 4, 15, 16, 0, 0, 0, time.Now().Location())
	require.NoError(t, persist(e, base, 1, 0, new(80.0), false))
	require.NoError(t, persist(e, base.Add(15*time.Minute), 1, 0, new(70.0), false))

	from := base.Add(-time.Hour).UTC()
	to := base.Add(time.Hour).UTC()

	// an aggregated bucket has no soc, a single slot carries its value
	res, err := QueryEnergy(from, to, "hour", false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Data, 1)
	require.Nil(t, res[0].Data[0].SocTemp)
	require.False(t, res[0].IsTemp) // battery: value is soc

	res, err = QueryEnergy(from, to, "15m", false)
	require.NoError(t, err)
	require.Len(t, res[0].Data, 2)
	require.Equal(t, 80.0, *res[0].Data[0].SocTemp)

	// grouped sums omit the per-entity snapshot
	res, err = QueryEnergy(from, to, "hour", true)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Nil(t, res[0].Data[0].SocTemp)
}

func TestQueryEnergyGridCost(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	grid := entity{Name: "grid", Group: Grid}
	require.NoError(t, db.Instance.Create(&grid).Error)
	pv := entity{Name: "pv", Group: PV}
	require.NoError(t, db.Instance.Create(&pv).Error)

	base := time.Date(2026, 4, 15, 16, 0, 0, 0, time.Now().Location())
	slot := func(i int) time.Time { return base.Add(time.Duration(i) * 15 * time.Minute) }

	// import 1 kWh at 0.20, 3 kWh at 0.40, 1 kWh unpriced; export 2 kWh at 0.10
	require.NoError(t, persist(grid, slot(0), 1, 2, nil, false))
	require.NoError(t, persist(grid, slot(1), 3, 0, nil, false))
	require.NoError(t, persist(grid, slot(2), 1, 0, nil, false))
	require.NoError(t, persist(pv, slot(0), 5, 0, nil, false))
	require.NoError(t, PersistTariffs(slot(0), new(0.2), new(0.1), nil, nil))
	require.NoError(t, PersistTariffs(slot(1), new(0.4), nil, nil, nil))

	res, err := QueryEnergy(base.Add(-time.Hour), base.Add(time.Hour), "hour", false)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// the bucket sums slot energy times slot price, unpriced slots stay out
	g := res[0].Data[0]
	require.Equal(t, Grid, res[0].Group)
	require.InDelta(t, 5, g.Energy, 1e-9)
	require.InDelta(t, 1.4, *g.Cost, 1e-9)
	require.InDelta(t, 4, *g.PricedEnergy, 1e-9)
	require.InDelta(t, 0.2, *g.ReturnCost, 1e-9)
	require.InDelta(t, 2, *g.PricedReturnEnergy, 1e-9)

	// other groups carry no cost
	require.Nil(t, res[1].Data[0].Cost)

	// a bucket starts at its boundary, not at the entity's first slot
	res, err = QueryEnergy(base.Add(-time.Hour), base.Add(time.Hour), "day", false)
	require.NoError(t, err)
	midnight := time.Date(2026, 4, 15, 0, 0, 0, 0, time.Now().Location())
	require.True(t, midnight.Equal(res[0].Data[0].Start))
	require.True(t, midnight.AddDate(0, 0, 1).Equal(res[0].Data[0].End))
}
