package metrics

import (
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
)

// DayClass is the classification of a completed day of an energy profile
type DayClass int

const (
	DayNormal     DayClass = iota
	DayAbsent              // consumption near base load all day, e.g. vacation
	DayOutlier             // consumption far above the typical day
	DayIncomplete          // missing or recovered slots, not classified
)

const (
	// absentDayFactor bounds the daily energy of an absent day as multiple of 24h base load
	absentDayFactor = 1.35
	// absentPeakFactor bounds every hour of an absent day as multiple of base load
	absentPeakFactor = 2.5
	// outlierDayFactor marks days above this multiple of the median daily energy
	outlierDayFactor = 2.0
	// nightHours are the hours of day [0, nightHours) used to derive the base load
	nightHours = 6
)

// dayEnergy holds the hourly energy of a single local calendar day
type dayEnergy struct {
	day    string      // local date YYYY-MM-DD
	hours  [24]float64 // energy per hour of day
	slots  int         // persisted slots
	broken bool        // contains recovered downtime energy
}

func (d dayEnergy) total() float64 {
	var sum float64
	for _, v := range d.hours {
		sum += v
	}
	return sum
}

func (d dayEnergy) peak() float64 {
	return slices.Max(d.hours[:])
}

// complete checks that every slot of the local day is persisted, DST days included
func (d dayEnergy) complete() bool {
	start, err := time.ParseInLocation(time.DateOnly, d.day, time.Local)
	if err != nil {
		return false
	}
	expected := int(start.AddDate(0, 0, 1).Sub(start) / tariff.SlotDuration)
	return !d.broken && d.slots >= expected
}

// median returns the median of values, 0 for an empty slice
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	s := slices.Clone(values)
	slices.Sort(s)

	if n := len(s); n%2 == 0 {
		return (s[n/2-1] + s[n/2]) / 2
	}
	return s[len(s)/2]
}

// classifyDays classifies complete days relative to the household's own base load,
// derived from the night hours of all complete days
func classifyDays(days []dayEnergy) map[string]DayClass {
	var nights, totals []float64
	for _, d := range days {
		if !d.complete() {
			continue
		}
		nights = append(nights, d.hours[:nightHours]...)
		totals = append(totals, d.total())
	}

	res := make(map[string]DayClass, len(days))

	base := median(nights)
	typical := median(totals)

	for _, d := range days {
		if !d.complete() {
			res[d.day] = DayIncomplete
			continue
		}

		if base <= 0 || typical <= 0 {
			res[d.day] = DayNormal
			continue
		}

		total := d.total()

		switch {
		case total < absentDayFactor*24*base && d.peak() < absentPeakFactor*base:
			res[d.day] = DayAbsent
		case total > outlierDayFactor*typical:
			res[d.day] = DayOutlier
		default:
			res[d.day] = DayNormal
		}
	}

	return res
}

// dailyEnergy queries the hourly energy per local day in [from, to)
func dailyEnergy(entity entity, from, to time.Time) ([]dayEnergy, error) {
	database, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// day as integer, the sqlite driver would scan a date string into a timestamp
	rows, err := database.Query(`SELECT
			CAST(strftime('%Y%m%d', ts, 'unixepoch', 'localtime') AS INTEGER) AS day,
			CAST(strftime('%H', ts, 'unixepoch', 'localtime') AS INTEGER) AS hour,
			SUM(COALESCE(energy, 0)),
			COUNT(*),
			MAX(COALESCE(recovered, 0))
		FROM meters
		WHERE meter = ? AND ts >= ? AND ts < ?
		GROUP BY day, hour
		ORDER BY day, hour`, entity.Id, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []dayEnergy
	for rows.Next() {
		var (
			ymd, hour, slots int
			energy           float64
			recovered        bool
		)

		if err := rows.Scan(&ymd, &hour, &energy, &slots, &recovered); err != nil {
			return nil, err
		}

		day := fmt.Sprintf("%04d-%02d-%02d", ymd/10000, ymd/100%100, ymd%100)

		if len(res) == 0 || res[len(res)-1].day != day {
			res = append(res, dayEnergy{day: day})
		}

		d := &res[len(res)-1]
		d.hours[hour] += energy
		d.slots += slots
		d.broken = d.broken || recovered
	}

	return res, rows.Err()
}

// ClassifyDays classifies the days with persisted energy in [from, to)
func (c *Collector) ClassifyDays(from, to time.Time) (map[string]DayClass, error) {
	days, err := dailyEnergy(c.entity, from, to)
	if err != nil {
		return nil, err
	}
	return classifyDays(days), nil
}
