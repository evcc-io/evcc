package metrics

import (
	"cmp"
	"errors"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/export"
	"gorm.io/gorm"
)

// Slot represents an aggregated energy time slot
type Slot struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Energy       float64   `json:"energy"`
	ReturnEnergy float64   `json:"returnEnergy"`
	SocTemp      *float64  `json:"socTemp,omitempty"` // a single slot's value
	// a price may not exist for every slot, priced energy is tracked separately to keep averages correct
	Cost               *float64 `json:"cost,omitempty"`               // grid import, where a price exists
	PricedEnergy       *float64 `json:"pricedEnergy,omitempty"`       // kWh with a price
	ReturnCost         *float64 `json:"returnCost,omitempty"`         // grid export, where a price exists
	PricedReturnEnergy *float64 `json:"pricedReturnEnergy,omitempty"` // kWh with a price
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

// localDate returns the first instant of a local date. That is not midnight
// where daylight saving skips it.
func localDate(year int, month time.Month, day int) time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	for noon := time.Date(year, month, day, 12, 0, 0, 0, time.Local); t.Day() != noon.Day(); {
		t = t.Add(tariff.SlotDuration)
	}
	return t
}

// localHour identifies the wall clock hour, which daylight saving can stretch
func localHour(t time.Time) [2]int {
	t = t.Local()
	return [2]int{t.YearDay(), t.Hour()}
}

// aggregateStarts aligns a slot to the start of its bucket, so all entities
// share the same bucket start. Buckets follow the local wall clock.
var aggregateStarts = map[string]func(time.Time) time.Time{
	"15m": func(t time.Time) time.Time { return t },
	"hour": func(t time.Time) time.Time {
		t = t.Truncate(tariff.SlotDuration)
		for prev := t.Add(-tariff.SlotDuration); localHour(prev) == localHour(t); prev = t.Add(-tariff.SlotDuration) {
			t = prev
		}
		return t.Local()
	},
	"day": func(t time.Time) time.Time {
		t = t.Local()
		return localDate(t.Year(), t.Month(), t.Day())
	},
	"month": func(t time.Time) time.Time {
		t = t.Local()
		return localDate(t.Year(), t.Month(), 1)
	},
}

// aggregateDurations returns the end of the bucket starting at t
var aggregateDurations = map[string]func(time.Time) time.Time{
	"15m": func(t time.Time) time.Time { return t.Add(tariff.SlotDuration) },
	"hour": func(t time.Time) time.Time {
		end := t.Add(tariff.SlotDuration)
		for localHour(end) == localHour(t) {
			end = end.Add(tariff.SlotDuration)
		}
		return end
	},
	"day":   func(t time.Time) time.Time { return localDate(t.Year(), t.Month(), t.Day()+1) },
	"month": func(t time.Time) time.Time { return localDate(t.Year(), t.Month()+1, 1) },
}

// EnergyFilter narrows QueryEnergy to matching entities. Empty fields are ignored.
type EnergyFilter struct {
	Group string
	Name  string
	Title string
}

// entityQuery returns a subquery selecting the ids of the matching entities,
// nil for an empty filter.
func entityQuery(f EnergyFilter) *gorm.DB {
	if f.Group == "" && f.Name == "" && f.Title == "" {
		return nil
	}

	tx := db.Instance.Model(new(entity)).Select("id")
	if f.Group != "" {
		tx = tx.Where(`"group" = ?`, f.Group)
	}
	if f.Name != "" {
		tx = tx.Where("name = ?", f.Name)
	}
	if f.Title != "" {
		tx = tx.Where("title = ?", f.Title)
	}

	return tx
}

// DeleteEnergy removes the slots in [from,to), narrowed to the matching
// entities. Both bounds are required, a full wipe is /api/db/reset.
func DeleteEnergy(from, to time.Time, filter ...EnergyFilter) (int64, error) {
	if from.IsZero() || to.IsZero() {
		return 0, errors.New("missing from/to")
	}

	var rows int64
	err := writeDays(from, to, func(tx *gorm.DB) error {
		tx = tx.Where("ts >= ? AND ts < ?", from.Unix(), to.Unix())

		if len(filter) > 0 {
			if sub := entityQuery(filter[0]); sub != nil {
				tx = tx.Where("meter IN (?)", sub)
			}
		}

		res := tx.Delete(new(meter))
		rows = res.RowsAffected
		return res.Error
	})
	if err != nil {
		return 0, err
	}

	return rows, nil
}

// QueryEnergy returns aggregated energy data, per title or per group.
func QueryEnergy(from, to time.Time, aggregate string, grouped bool, filter ...EnergyFilter) ([]Series, error) {
	addDuration := aggregateDurations[aggregate]
	bucketStart, ok := aggregateStarts[aggregate]
	if !ok {
		return nil, errors.New("invalid aggregate value")
	}

	var ents []entity
	tx := db.Instance.Select(`id, "group", COALESCE(NULLIF(title,''), name) AS title, is_temp`)
	var ids []int
	if len(filter) > 0 {
		if sub := entityQuery(filter[0]); sub != nil {
			tx = tx.Where("id IN (?)", sub)
			ids = []int{}
		}
	}
	if err := tx.Find(&ents).Error; err != nil {
		return nil, err
	}

	entities := make(map[int]entity, len(ents))
	for _, e := range ents {
		entities[e.Id] = e
		if ids != nil {
			ids = append(ids, e.Id)
		}
	}

	from, to, err := closeBounds(from, to, slotRange)
	if err != nil {
		return nil, err
	}

	var sums []energySum
	switch aggregate {
	case "15m":
		sums, err = sumEnergy([][2]int64{{from.Unix(), to.Unix()}}, true, ids)
	case "hour":
		sums, err = sumEnergy(buckets(from, to, aggregate), false, ids)
	default:
		sums, err = dailyEnergy(from, to, ids)
	}
	if err != nil {
		return nil, err
	}

	// meters sharing a title or group are one series, days add up to months
	type key struct {
		title, group string
		start        int64
	}
	series := make(map[key]*energySum)
	var keys []key

	add := func(dst **float64, src *float64) {
		if src == nil {
			return
		}
		if *dst == nil {
			*dst = new(float64)
		}
		**dst += *src
	}

	for _, s := range sums {
		e := entities[s.Meter]
		k := key{e.Title, e.Group, bucketStart(time.Unix(s.Start, 0)).Unix()}
		if grouped {
			k.title = ""
		}

		b := series[k]
		if b == nil {
			b = &energySum{Meter: s.Meter, SocTemp: s.SocTemp}
			series[k] = b
			keys = append(keys, k)
		}

		b.Slots += s.Slots
		b.Energy += s.Energy
		b.ReturnEnergy += s.ReturnEnergy
		add(&b.Cost, s.Cost)
		add(&b.PricedEnergy, s.PricedEnergy)
		add(&b.ReturnCost, s.ReturnCost)
		add(&b.PricedReturnEnergy, s.PricedReturnEnergy)
	}

	slices.SortFunc(keys, func(a, b key) int {
		return cmp.Or(cmp.Compare(a.title, b.title), cmp.Compare(a.group, b.group), cmp.Compare(a.start, b.start))
	})

	var res []Series
	for _, k := range keys {
		b := series[k]

		if n := len(res); n == 0 || res[n-1].Title != k.title || res[n-1].Group != k.group {
			res = append(res, Series{Title: k.title, Group: k.group, IsTemp: !grouped && entities[b.Meter].IsTemp})
		}

		s := &res[len(res)-1]
		start := time.Unix(k.start, 0)
		slot := Slot{
			Start:        start,
			End:          addDuration(start),
			Energy:       roundEnergy(b.Energy),
			ReturnEnergy: roundEnergy(b.ReturnEnergy),

			Cost:               b.Cost,
			PricedEnergy:       b.PricedEnergy,
			ReturnCost:         b.ReturnCost,
			PricedReturnEnergy: b.PricedReturnEnergy,
		}
		// soc_temp only for a single slot bucket; omitted for grouped sums
		if !grouped && b.Slots == 1 {
			slot.SocTemp = b.SocTemp
		}
		s.Data = append(s.Data, slot)
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
