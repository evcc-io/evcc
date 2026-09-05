package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
)

var ErrIncomplete = errors.New("meter profile incomplete")

// energyProfileFiltered queries the 96-slot profile, optionally restricted to a
// single weekday (strftime %w, 0=Sunday).
func energyProfileFiltered(entity entity, from time.Time, weekday *int) (*[96]float64, error) {
	database, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	args := []any{entity.Id, from.Unix()}

	var weekdayFilter string
	if weekday != nil {
		// CAST is required, strftime returns TEXT which never compares equal to an integer
		weekdayFilter = ` AND CAST(strftime('%w', ts, 'unixepoch', 'localtime') AS INTEGER) = ?`
		args = append(args, *weekday)
	}

	// COALESCE guards against legacy rows with NULL energy
	rows, err := database.Query(`SELECT min(ts) AS ts, COALESCE(avg(energy), 0) AS energy
		FROM meters
		WHERE meter = ? AND ts >= ? AND COALESCE(recovered, 0) = 0`+weekdayFilter+`
		GROUP BY strftime('%H:%M', ts, 'unixepoch', 'localtime')
		ORDER BY strftime('%H:%M', ts, 'unixepoch', 'localtime') ASC`,
		args...,
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
