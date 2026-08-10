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

// energyProfileWeekday returns a 96-slot 15min average energy profile (kWh) for the
// same weekday as today, averaged across the past 4 occurrences (28 days).
// Groups by time-of-day slot. Returns ErrIncomplete if fewer than 96 slots are present.
func energyProfileWeekday(entity entity) (*[96]float64, error) {
	database, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	weekdayNum := int(time.Now().Weekday()) // 0=Sunday
	from := time.Now().AddDate(0, 0, -28)
	rows, err := database.Query(`SELECT min(ts) AS ts, COALESCE(avg(energy), 0) AS energy
		FROM meters
		WHERE meter = ? AND ts >= ? AND COALESCE(recovered, 0) = 0
		  AND strftime('%w', ts, 'unixepoch', 'localtime') = ?
		GROUP BY strftime("%H:%M", ts, 'unixepoch', 'localtime')
		ORDER BY strftime("%H:%M", ts, 'unixepoch', 'localtime') ASC`,
		entity.Id, from.Unix(), weekdayNum,
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
