package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meter_ts"`
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meter_ts"`
	Value     float64   `json:"val" gorm:"column:val"`
}

var ErrIncomplete = errors.New("meter profile incomplete")

func Init() error {
	return db.Instance.AutoMigrate(new(meter))
}

// Persist stores 15min consumption in Wh
func Persist(ts time.Time, value float64) error {
	return db.Instance.Create(meter{
		Meter:     1,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// Profile returns a 15min average meter profile in Wh.
// Profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func Profile(from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// Use 'localtime' in strftime to fix https://github.com/evcc-io/evcc/discussions/23759
	rows, err := db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ? AND ts >= ?
		GROUP BY strftime("%H:%M", ts, 'localtime')
		ORDER BY strftime("%H:%M", ts, 'localtime') ASC`, 1, from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]float64, 0, 96)

	for rows.Next() {
		var ts SqlTime
		var val float64

		if err := rows.Scan(&ts, &val); err != nil {
			return nil, err
		}

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

func SlotNum(ts time.Time) int {
	ts = ts.Local()
	return ts.Hour()*4 + ts.Minute()/15
}
