package metrics

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

// Slot represents an aggregated energy time slot
type Slot struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Energy       float64   `json:"energy"`
	ReturnEnergy float64   `json:"returnEnergy"`
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
			Energy:       r.Energy,
			ReturnEnergy: r.ReturnEnergy,
		})
	}

	return res, nil
}
