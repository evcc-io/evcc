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

func Init() error {
	return db.Instance.AutoMigrate(new(meter))
}

// Persist stores 15min consumption in Wh
func Persist(ts time.Time, value float64) error {
	return db.Instance.Save(meter{
		Meter:     1,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// Profile returns a 15min average meter profile in Wh
func Profile() ([96]float64, error) {
	var res [96]float64

	db, err := db.Instance.DB()
	if err != nil {
		return res, err
	}

	rows, err := db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ?
		GROUP BY strftime("HH:MM", ts)
		ORDER BY ts`, 1)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	var i int

	for rows.Next() {
		var ts time.Time
		if err := rows.Scan(&ts, &res[i]); err != nil {
			return res, err
		}

		hour := ts.Hour()
		minute := ts.Minute() / 15
		if i != hour*4+minute {
			return res, errors.New("meter profile incomplete")
		}

		i++
	}

	if i != 96 {
		return res, errors.New("meter profile incomplete")
	}

	return res, nil
}
