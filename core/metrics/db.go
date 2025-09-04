package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meter_ts"`
	Entity    entity    `json:"-" gorm:"foreignkey:Meter;references:Id"`
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meter_ts"`
	Value     float64   `json:"val" gorm:"column:val"`
}

type entity struct {
	Id   int    `gorm:"column:id;primarykey"`
	Name string `gorm:"column:name;uniqueIndex:name_idx"`
}

var ErrIncomplete = errors.New("meter profile incomplete")

func Init() error {
	hasTable := db.Instance.Migrator().HasTable("metrics")

	// create entity first to make sure foreign keys for existing data work
	if err := db.Instance.AutoMigrate(new(entity)); err != nil {
		return err
	}

	// create entity for id 1
	if !hasTable {
		if _, err := createEntity(Home); err != nil {
			return err
		}
	}

	return db.Instance.AutoMigrate(new(meter))
}

// persist stores 15min consumption in Wh
func persist(entity entity, ts time.Time, value float64) error {
	return db.Instance.Create(&meter{
		Entity:    entity,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// profile returns a 15min average meter profile in Wh.
// profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func profile(entity entity, from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ? AND ts >= ?
		GROUP BY strftime("%H:%M", ts)
		ORDER BY strftime("%H:%M", ts) ASC`, 1, from,
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
