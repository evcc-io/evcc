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

// Profile returns a 15min average meter profile in Wh
func Profile() (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ?
		GROUP BY strftime("%H:%M", ts)
		ORDER BY ts`, 1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]float64, 0, 96)

	for rows.Next() {
		var (
			ts  time.Time
			val float64
		)
		if err := rows.Scan(&ts, &val); err != nil {
			return nil, err
		}

		hour := ts.Hour()
		minute := ts.Minute() / 15
		if len(res) != hour*4+minute {
			return nil, ErrIncomplete
		}

		res = append(res, val)
	}

	if len(res) != 96 {
		return nil, ErrIncomplete
	}

	return (*[96]float64)(res), nil
}
