package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
)

var ErrIncomplete = errors.New("meter profile incomplete")

// energyProfile returns a 15min average meter profile in kWh, averaged across all
// days in [from, now). Groups by time-of-day (96 slots). Returns ErrIncomplete if
// fewer than 96 slots are present.
func energyProfile(entity entity, from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// COALESCE guards against legacy rows with NULL energy
	rows, err := db.Query(`SELECT min(ts) AS ts, COALESCE(avg(energy), 0) AS energy
		FROM meters
		WHERE meter = ? AND ts >= ? AND COALESCE(recovered, 0) = 0
		GROUP BY strftime("%H:%M", ts, 'unixepoch', 'localtime')
		ORDER BY strftime("%H:%M", ts, 'unixepoch', 'localtime') ASC`,
		entity.Id, from.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prev time.Time
	res := make([]float64, 0, 96)

	for rows.Next() {
		var ts SqlTime
		var val float64

		if err := rows.Scan(&ts, &val); err != nil {
			return nil, err
		}

		// interpolate single missing value, maybe due to regular restarts?
		if time.Time(ts).Sub(prev) == 2*tariff.SlotDuration {
			res = append(res, (val+res[len(res)-1])/2)
		}
		prev = time.Time(ts)

		res = append(res, val)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(res) != 96 {
		return nil, ErrIncomplete
	}

	return (*[96]float64)(res), nil
}

// energyProfileWeekday returns the actual 96-slot 15min energy profile (kWh) for the
// same weekday as now, taken from 7 days ago. No averaging — one day is the forecast.
// Returns ErrIncomplete if fewer than 96 slots are present for that day.
func energyProfileWeekday(entity entity) (*[96]float64, error) {
	database, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// same weekday, 7 days back: covers exactly 00:00–23:45 of that day
	weekdayNum := int(time.Now().Weekday()) // 0=Sunday
	rows, err := database.Query(`SELECT ts, COALESCE(energy, 0) AS energy
		FROM meters
		WHERE meter = ? AND COALESCE(recovered, 0) = 0
		  AND strftime('%w', ts, 'unixepoch', 'localtime') = ?
		  AND ts >= ?
		  AND ts < ?
		ORDER BY ts ASC`,
		entity.Id,
		weekdayNum,
		time.Now().AddDate(0, 0, -7).Truncate(24*time.Hour).Unix(),
		time.Now().AddDate(0, 0, -6).Truncate(24*time.Hour).Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prev time.Time
	res := make([]float64, 0, 96)

	for rows.Next() {
		var ts SqlTime
		var val float64

		if err := rows.Scan(&ts, &val); err != nil {
			return nil, err
		}

		// interpolate single missing value
		if !prev.IsZero() && time.Time(ts).Sub(prev) == 2*tariff.SlotDuration {
			res = append(res, (val+res[len(res)-1])/2)
		}
		prev = time.Time(ts)

		res = append(res, val)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(res) != 96 {
		return nil, ErrIncomplete
	}

	return (*[96]float64)(res), nil
}
