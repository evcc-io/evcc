package metrics

import (
	"database/sql"
	"errors"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
)

var ErrIncomplete = errors.New("meter profile incomplete")

// profilePercentile returns the configured per-slot percentile of the energy profiles (0..1), 0 = average
func profilePercentile() float64 {
	v, err := settings.Float(keys.ProfilePercentile)
	if err != nil {
		return 0
	}
	return v / 100
}

// energyProfileFiltered queries the 96-slot profile at the given percentile
// (0..1, linear interpolation between ranks; 0 = average), optionally
// restricted to a single weekday (strftime %w, 0=Sunday).
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

	// COALESCE guards against legacy rows with NULL energy
	slots := `SELECT ts, COALESCE(energy, 0) AS energy, strftime('%H:%M', ts, 'unixepoch', 'localtime') AS slot
		FROM meters
		WHERE meter = ? AND ts >= ? AND COALESCE(recovered, 0) = 0` + weekdayFilter

	query := `WITH slots AS (` + slots + `)
		SELECT min(ts) AS ts, avg(energy) AS energy
		FROM slots
		GROUP BY slot
		ORDER BY slot ASC`

	if percentile > 0 {
		// rank each slot's values and weight the two ranks enclosing the percentile
		// position (1-based, pos = p * (n-1) + 1) by their distance to it.
		// The slots CTE must stay first to keep the placeholder order.
		args = append(args, percentile)
		query = `WITH slots AS (` + slots + `), ranked AS (
			SELECT slot, energy,
				min(ts) OVER (PARTITION BY slot) AS ts,
				row_number() OVER (PARTITION BY slot ORDER BY energy) AS rn,
				? * (count(*) OVER (PARTITION BY slot) - 1) + 1 AS pos
			FROM slots
		)
		SELECT ts, sum(energy * (1 - abs(rn - pos))) AS energy
		FROM ranked
		WHERE abs(rn - pos) < 1
		GROUP BY slot
		ORDER BY slot ASC`
	}

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProfile(rows)
}

// energyProfileTemperatureBinned queries the historical average energy per 15-minute slot
// and rounded temperature (degree Celsius).
func energyProfileTemperatureBinned(entity entity) (map[int]map[int]float64, error) {
	database, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	from := now.BeginningOfDay().AddDate(-1, 0, 0).Unix()
	before := now.BeginningOfDay().Unix()
	query := `SELECT (CAST(strftime('%H', m.ts, 'unixepoch', 'localtime') AS INTEGER) * 4 +
		CAST(strftime('%M', m.ts, 'unixepoch', 'localtime') AS INTEGER) / 15) AS slot_idx,
		CAST(round(t.temperature) AS INTEGER) AS temp_bin,
		avg(m.energy) AS energy
	FROM meters m
	JOIN tariffs t ON m.ts = t.ts
	WHERE m.meter = ? AND m.ts >= ? AND m.ts < ? AND COALESCE(m.recovered, 0) = 0
		AND t.temperature IS NOT NULL
	GROUP BY slot_idx, temp_bin
	HAVING count(*) >= 2`

	rows, err := database.Query(query, entity.Id, from, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int]map[int]float64)

	for rows.Next() {
		var slotIdx int
		var tempBin int
		var energy float64

		if err := rows.Scan(&slotIdx, &tempBin, &energy); err != nil {
			return nil, err
		}

		if slotIdx < 0 || slotIdx >= 96 {
			continue
		}

		if res[slotIdx] == nil {
			res[slotIdx] = make(map[int]float64)
		}
		res[slotIdx][tempBin] = energy
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, ErrIncomplete
	}

	return res, nil
}

func scanProfile(rows *sql.Rows) (*[96]float64, error) {
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
