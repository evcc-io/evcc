package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
)

var ErrIncomplete = errors.New("meter profile incomplete")

// profilePercentile selects the per-slot percentile of the energy profile.
// 0.5 is the median, which keeps a few heavy days from dominating the average.
const profilePercentile = 0.5

// energyProfileFiltered queries the 96-slot profile at the given percentile
// (0..1, linear interpolation between ranks), optionally restricted to a
// single weekday (strftime %w, 0=Sunday).
func energyProfileFiltered(entity entity, from time.Time, weekday *int, percentile float64) (*[96]float64, error) {
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
	args = append(args, percentile)

	// rank each slot's values and interpolate between the two ranks enclosing
	// the percentile position (1-based, pos = p * (n-1) + 1)
	// COALESCE guards against legacy rows with NULL energy
	rows, err := database.Query(`WITH slots AS (
			SELECT ts, COALESCE(energy, 0) AS energy, strftime('%H:%M', ts, 'unixepoch', 'localtime') AS slot
			FROM meters
			WHERE meter = ? AND ts >= ? AND COALESCE(recovered, 0) = 0`+weekdayFilter+`
		), ranked AS (
			SELECT slot, energy,
				min(ts) OVER (PARTITION BY slot) AS ts,
				row_number() OVER (PARTITION BY slot ORDER BY energy) AS rn,
				? * (count(*) OVER (PARTITION BY slot) - 1) + 1 AS pos
			FROM slots
		)
		SELECT ts, sum(energy * CASE WHEN rn = CAST(pos AS INTEGER) THEN 1 - pos + CAST(pos AS INTEGER) ELSE pos - CAST(pos AS INTEGER) END) AS energy
		FROM ranked
		WHERE rn BETWEEN CAST(pos AS INTEGER) AND CAST(pos AS INTEGER) + 1
		GROUP BY slot
		ORDER BY slot ASC`,
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
