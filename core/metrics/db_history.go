package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

// Slot represents an aggregated energy time slot
type Slot struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Import float64   `json:"import"`
	Export float64   `json:"export"`
}

// Series represents a named series of energy slots
type Series struct {
	Name  string `json:"name,omitempty"`
	Group string `json:"group"`
	Data  []Slot `json:"data"`
}

var aggregateFormats = map[string]string{
	"15m":   "%Y-%m-%d %H:%M",
	"hour":  "%Y-%m-%d %H:00",
	"day":   "%Y-%m-%d",
	"month": "%Y-%m",
}

var aggregateGoFormats = map[string]string{
	"15m":   "2006-01-02 15:04",
	"hour":  "2006-01-02 15:00",
	"day":   "2006-01-02",
	"month": "2006-01",
}

var aggregateDurations = map[string]func(time.Time) time.Time{
	"15m":   func(t time.Time) time.Time { return t.Add(15 * time.Minute) },
	"hour":  func(t time.Time) time.Time { return t.Add(time.Hour) },
	"day":   func(t time.Time) time.Time { return t.AddDate(0, 0, 1) },
	"month": func(t time.Time) time.Time { return t.AddDate(0, 1, 0) },
}

// QueryImportEnergy returns aggregated energy data, per entity or per group.
func QueryImportEnergy(from, to time.Time, aggregate string, grouped bool) ([]Series, error) {
	format, ok := aggregateFormats[aggregate]
	if !ok {
		return nil, errors.New("invalid aggregate value")
	}

	addDuration := aggregateDurations[aggregate]

	groupCols := `e.name, e."group", bucket`
	if grouped {
		groupCols = `e."group", bucket`
	}

	type row struct {
		Name   string
		Group  string
		Bucket string
		Import float64
		Export float64
	}

	tx := db.Instance.Table("meters m").
		Select(`e.name, e."group", strftime(?, m.ts, 'unixepoch', 'localtime') AS bucket,
			COALESCE(SUM(m."import"), 0) AS import, COALESCE(SUM(m.export), 0) AS export`, format).
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
		start, err := time.ParseInLocation(aggregateGoFormats[aggregate], r.Bucket, from.Location())
		if err != nil {
			return nil, err
		}

		name := r.Name
		if grouped {
			name = ""
		}

		if n := len(res); n == 0 || res[n-1].Name != name || res[n-1].Group != r.Group {
			res = append(res, Series{Name: name, Group: r.Group})
		}

		s := &res[len(res)-1]
		s.Data = append(s.Data, Slot{
			Start:  start,
			End:    addDuration(start),
			Import: r.Import,
			Export: r.Export,
		})
	}

	return res, nil
}
