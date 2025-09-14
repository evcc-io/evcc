package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meters_meter_ts"`
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meters_meter_ts"`
	Entity    entity    `json:"-" gorm:"foreignkey:Meter;references:Id"`
	Import    float64   `json:"import" gorm:"column:import"`
	Export    float64   `json:"export" gorm:"column:export"`
}

type entity struct {
	Id    int    `gorm:"column:id;primarykey"`
	Group string `gorm:"column:group;uniqueIndex:entities_group_name"`
	Name  string `gorm:"column:name;uniqueIndex:entities_group_name"`
}

var ErrIncomplete = errors.New("meter profile incomplete")

func Init() error {
	m := db.Instance.Migrator()

	// entites: create entity first to make sure foreign keys for existing data work
	hasTable := m.HasTable(new(entity))
	if err := db.Instance.AutoMigrate(new(entity)); err != nil {
		return err
	}

	// entites: add entity id 1
	if hasTable {
		var res entity
		if err := db.Instance.Where(&entity{Id: 1, Name: Home}).FirstOrCreate(&res).Error; err != nil {
			return err
		}

		if res.Group == "" {
			res.Group = Virtual
			if err := db.Instance.Save(&res).Error; err != nil {
				return err
			}
		}
	}

	// drop obsolete indexes
	for _, idx := range []struct {
		name string
		obj  any
	}{
		{"name_idx", new(entity)},
		{"group_name", new(entity)},
		{"meter_ts", new(meter)},
	} {
		if m.HasIndex(idx.obj, idx.name) {
			if err := m.DropIndex(idx.obj, idx.name); err != nil {
				return err
			}
		}
	}

	// meter: split energy direction
	if old := "val"; m.HasColumn(new(meter), old) {
		if err := m.RenameColumn(new(meter), old, "pos"); err != nil {
			return err
		}
	}

	return db.Instance.AutoMigrate(new(meter))
}

// persist stores 15min consumption in Wh
func persist(entity entity, ts time.Time, imprt, export float64) error {
	return db.Instance.Create(&meter{
		Entity:    entity,
		Timestamp: ts.Truncate(15 * time.Minute),
		Import:    imprt,
		Export:    export,
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
