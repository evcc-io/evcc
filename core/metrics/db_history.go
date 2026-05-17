package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/server/db"
	csvutil "github.com/evcc-io/evcc/util/csv"
)

// Slot represents an aggregated energy time slot
type Slot struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Energy       float64   `json:"energy"`
	ReturnEnergy float64   `json:"returnEnergy"`
}

// roundEnergy rounds kWh to Wh precision and clamps negative noise to zero.
func roundEnergy(v float64) float64 {
	return max(0, math.Round(v*1000)/1000)
}

// Series represents a named series of energy slots
type Series struct {
	Name  string `json:"name,omitempty"`
	Group string `json:"group"`
	Data  []Slot `json:"data"`
}

// SeriesCSV wraps a slice of Series for CSV export.
type SeriesCSV []Series

// csvGroupOrder mirrors the frontend GROUP_ORDER plus home/forecast.
var csvGroupOrder = []string{PV, Battery, Grid, Loadpoint, Meter, Home, Forecast}

var aggregateFormats = map[string]string{
	"15m":   "%Y-%m-%d %H:%M",
	"hour":  "%Y-%m-%d %H:00",
	"day":   "%Y-%m-%d",
	"month": "%Y-%m",
}

var aggregateDurations = map[string]func(time.Time) time.Time{
	"15m":   func(t time.Time) time.Time { return t.Add(15 * time.Minute) },
	"hour":  func(t time.Time) time.Time { return t.Add(time.Hour) },
	"day":   func(t time.Time) time.Time { return t.AddDate(0, 0, 1) },
	"month": func(t time.Time) time.Time { return t.AddDate(0, 1, 0) },
}

// QueryEnergy returns aggregated energy data, per entity or per group.
func QueryEnergy(from, to time.Time, aggregate string, grouped bool) ([]Series, error) {
	addDuration := aggregateDurations[aggregate]

	format, ok := aggregateFormats[aggregate]
	if !ok {
		return nil, errors.New("invalid aggregate value")
	}

	groupCols := `e."group", ` + fmt.Sprintf(`strftime('%s', m.ts, 'unixepoch', 'localtime')`, format)
	if !grouped {
		groupCols = "e.name, " + groupCols
	}

	type row struct {
		Name         string
		Group        string
		Start        SqlTime
		Energy       float64
		ReturnEnergy float64
	}

	tx := db.Instance.Table("meters m").
		Select(`e.name, e."group",
			MIN(m.ts) AS start,
			COALESCE(SUM(m.energy), 0) AS energy,
			COALESCE(SUM(m.return_energy), 0) AS return_energy`).
		Joins("JOIN entities e ON m.meter = e.id").
		Group(groupCols).
		Order(groupCols)

	if !from.IsZero() {
		tx = tx.Where("m.ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("m.ts < ?", to.Unix())
	}

	var rows []row
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}

	var res []Series
	for _, r := range rows {
		name := r.Name
		if grouped {
			name = ""
		}

		if n := len(res); n == 0 || res[n-1].Name != name || res[n-1].Group != r.Group {
			res = append(res, Series{Name: name, Group: r.Group})
		}

		s := &res[len(res)-1]
		s.Data = append(s.Data, Slot{
			Start:        time.Time(r.Start),
			End:          addDuration(time.Time(r.Start)),
			Energy:       roundEnergy(r.Energy),
			ReturnEnergy: roundEnergy(r.ReturnEnergy),
		})
	}

	return res, nil
}

// hasReturnEnergy reports whether a group's CSV export includes a returnEnergy
// column. Only the bidirectional groups (grid, battery) do; the rest emit a
// single energy column.
func hasReturnEnergy(group string) bool {
	return group == Grid || group == Battery
}

// WriteCsv emits a wide-table CSV with columns
//
//	time.start, time.end, <group>.<entity>.energy.Wh[, <group>.<entity>.returnEnergy.Wh], …
//
// Only grid and battery contribute a second returnEnergy column.
// Values are plain Wh integers (the DB is already rounded to milli-kWh) — no
// decimal point, no thousands separator, so they survive locale-mismatched
// spreadsheet importers unambiguously.
func (s SeriesCSV) WriteCsv(ctx context.Context, w io.Writer) error {
	ww, _, err := csvutil.NewLocalizedWriter(ctx, w)
	if err != nil {
		return err
	}

	byGroup := make(map[string][]*Series)
	for i := range s {
		g := s[i].Group
		byGroup[g] = append(byGroup[g], &s[i])
	}

	rank := make(map[string]int, len(csvGroupOrder))
	for i, g := range csvGroupOrder {
		rank[g] = i
	}
	groups := make([]string, 0, len(byGroup))
	for g := range byGroup {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		ri, oki := rank[groups[i]]
		rj, okj := rank[groups[j]]
		switch {
		case oki && okj:
			return ri < rj
		case oki:
			return true
		case okj:
			return false
		default:
			return groups[i] < groups[j]
		}
	})

	header := []string{"time.start", "time.end"}
	type col struct {
		series       *Series
		returnEnergy bool
	}
	cols := []col{{}, {}}
	tsSet := make(map[int64]time.Time)
	endByStart := make(map[int64]time.Time)

	for _, g := range groups {
		entities := byGroup[g]
		sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })

		for _, e := range entities {
			name := e.Name
			if name == "" {
				name = g
			}
			header = append(header, g+"."+name+".energy.Wh")
			cols = append(cols, col{series: e, returnEnergy: false})
			if hasReturnEnergy(g) {
				header = append(header, g+"."+name+".returnEnergy.Wh")
				cols = append(cols, col{series: e, returnEnergy: true})
			}
			for _, slot := range e.Data {
				tsSet[slot.Start.UnixNano()] = slot.Start
				endByStart[slot.Start.UnixNano()] = slot.End
			}
		}
	}

	if err := ww.Write(header); err != nil {
		return err
	}

	if len(tsSet) == 0 {
		ww.Flush()
		return ww.Error()
	}

	timestamps := make([]time.Time, 0, len(tsSet))
	for _, t := range tsSet {
		timestamps = append(timestamps, t)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i].Before(timestamps[j]) })

	slotIdx := make(map[*Series]map[int64]Slot)
	for _, list := range byGroup {
		for _, e := range list {
			m := make(map[int64]Slot, len(e.Data))
			for _, slot := range e.Data {
				m[slot.Start.UnixNano()] = slot
			}
			slotIdx[e] = m
		}
	}

	row := make([]string, len(cols))
	for _, ts := range timestamps {
		row[0] = ts.Local().Format("2006-01-02 15:04:05")
		row[1] = endByStart[ts.UnixNano()].Local().Format("2006-01-02 15:04:05")
		for i := 2; i < len(cols); i++ {
			c := cols[i]
			if c.series == nil {
				row[i] = ""
				continue
			}
			slot, ok := slotIdx[c.series][ts.UnixNano()]
			if !ok {
				row[i] = ""
				continue
			}
			v := slot.Energy
			if c.returnEnergy {
				v = slot.ReturnEnergy
			}
			row[i] = strconv.FormatInt(int64(math.Round(v*1000)), 10)
		}
		if err := ww.Write(row); err != nil {
			return err
		}
	}

	ww.Flush()
	return ww.Error()
}
