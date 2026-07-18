package metrics

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/export"
)

// Slot represents an aggregated energy time slot
type Slot struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Energy       float64   `json:"energy"`
	ReturnEnergy float64   `json:"returnEnergy"`
	SocTemp      *float64  `json:"socTemp,omitempty"`
}

// roundEnergy rounds kWh to Wh precision and clamps negative noise to zero.
func roundEnergy(v float64) float64 {
	return max(0, math.Round(v*1000)/1000)
}

// Series represents an energy series for one title group or one entity group.
type Series struct {
	Title  string `json:"title,omitempty"`
	Group  string `json:"group"`
	IsTemp bool   `json:"isTemp,omitempty"` // socTemp values are temperature, not soc
	Data   []Slot `json:"data"`
}

// SeriesExport wraps a slice of Series for tabular export.
type SeriesExport []Series

// GroupOrder is the canonical display order of metric groups, mirroring the
// frontend GROUP_ORDER plus home/forecast.
var GroupOrder = []string{PV, Battery, Grid, Loadpoint, Consumer, Meter, Home, Forecast, Temperature}

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

// EnergyFilter narrows QueryEnergy to matching entities. Empty fields are ignored.
type EnergyFilter struct {
	Group string
	Name  string
	Title string
}

// QueryEnergy returns aggregated energy data, per title or per group.
func QueryEnergy(from, to time.Time, aggregate string, grouped bool, filter ...EnergyFilter) ([]Series, error) {
	addDuration := aggregateDurations[aggregate]

	format, ok := aggregateFormats[aggregate]
	if !ok {
		return nil, errors.New("invalid aggregate value")
	}

	titleExpr := `COALESCE(NULLIF(e.title,''), e.name)`
	timeCol := fmt.Sprintf(`strftime('%s', m.ts, 'unixepoch', 'localtime')`, format)

	selectTitle := titleExpr + ` AS title`
	groupCols := titleExpr + `, e."group", ` + timeCol
	if grouped {
		selectTitle = `'' AS title`
		groupCols = `e."group", ` + timeCol
	}

	type row struct {
		Title        string
		Group        string
		Start        SqlTime
		Energy       float64
		ReturnEnergy float64
		SocTemp      *float64
		IsTemp       bool
	}

	// soc_temp reports the bucket's first slot; omitted for grouped sums
	socCols := `, m.soc_temp AS soc_temp, e.is_temp AS is_temp`
	if grouped {
		socCols = ``
	}

	tx := db.Instance.Table("meters m").
		Select(selectTitle + `, e."group",
			MIN(m.ts) AS start,
			COALESCE(SUM(m.energy), 0) AS energy,
			COALESCE(SUM(m.return_energy), 0) AS return_energy` + socCols).
		Joins("JOIN entities e ON m.meter = e.id").
		Group(groupCols).
		Order(groupCols)

	if !from.IsZero() {
		tx = tx.Where("m.ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("m.ts < ?", to.Unix())
	}

	if len(filter) > 0 {
		f := filter[0]
		if f.Group != "" {
			tx = tx.Where(`e."group" = ?`, f.Group)
		}
		if f.Name != "" {
			tx = tx.Where("e.name = ?", f.Name)
		}
		if f.Title != "" {
			tx = tx.Where("e.title = ?", f.Title)
		}
	}

	var rows []row
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}

	var res []Series
	for _, r := range rows {
		if n := len(res); n == 0 || res[n-1].Title != r.Title || res[n-1].Group != r.Group {
			res = append(res, Series{Title: r.Title, Group: r.Group, IsTemp: r.IsTemp})
		}

		s := &res[len(res)-1]
		s.Data = append(s.Data, Slot{
			Start:        time.Time(r.Start),
			End:          addDuration(time.Time(r.Start)),
			Energy:       roundEnergy(r.Energy),
			ReturnEnergy: roundEnergy(r.ReturnEnergy),
			SocTemp:      r.SocTemp,
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

func seriesHasSocTemp(s *Series) bool {
	return slices.ContainsFunc(s.Data, func(slot Slot) bool { return slot.SocTemp != nil })
}

// socTempValue rounds a soc/temp value to 0.1, nil when unset
func socTempValue(v *float64) any {
	if v == nil {
		return nil
	}
	return math.Round(*v*10) / 10
}

// Write emits the wide table to ww: time.start, time.end, then one energy.Wh
// column per entity (grid/battery add returnEnergy.Wh) as plain locale-safe Wh ints.
func (s SeriesExport) Write(ww export.RowWriter) error {
	byGroup := make(map[string][]*Series)
	for i := range s {
		g := s[i].Group
		byGroup[g] = append(byGroup[g], &s[i])
	}

	rank := make(map[string]int, len(GroupOrder))
	for i, g := range GroupOrder {
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

	header := []any{"time.start", "time.end"}
	type col struct {
		series       *Series
		returnEnergy bool
		socTemp      bool
	}
	cols := []col{{}, {}}
	tsSet := make(map[int64]time.Time)
	endByStart := make(map[int64]time.Time)

	label := func(e *Series, g string) string {
		if e.Title != "" {
			return e.Title
		}
		return g
	}

	prefix := func(g, l string) string {
		if l == g {
			return g
		}
		return g + "." + l
	}

	for _, g := range groups {
		entities := byGroup[g]
		sort.Slice(entities, func(i, j int) bool { return label(entities[i], g) < label(entities[j], g) })

		for _, e := range entities {
			p := prefix(g, label(e, g))
			header = append(header, p+".energy.Wh")
			cols = append(cols, col{series: e, returnEnergy: false})
			if hasReturnEnergy(g) {
				header = append(header, p+".returnEnergy.Wh")
				cols = append(cols, col{series: e, returnEnergy: true})
			}
			if seriesHasSocTemp(e) {
				unit := ".soc.pct"
				if e.IsTemp {
					unit = ".temp.degC"
				}
				header = append(header, p+unit)
				cols = append(cols, col{series: e, socTemp: true})
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

	row := make([]any, len(cols))
	for _, ts := range timestamps {
		row[0] = ts.Local()
		row[1] = endByStart[ts.UnixNano()].Local()
		for i := 2; i < len(cols); i++ {
			c := cols[i]
			if c.series == nil {
				row[i] = nil
				continue
			}
			slot, ok := slotIdx[c.series][ts.UnixNano()]
			if !ok {
				row[i] = nil
				continue
			}
			switch {
			case c.socTemp:
				row[i] = socTempValue(slot.SocTemp)
			case c.returnEnergy:
				row[i] = int64(math.Round(slot.ReturnEnergy * 1000))
			default:
				row[i] = int64(math.Round(slot.Energy * 1000))
			}
		}
		if err := ww.Write(row); err != nil {
			return err
		}
	}

	ww.Flush()
	return ww.Error()
}
