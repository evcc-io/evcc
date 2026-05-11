package metrics

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

type meter struct {
	Meter     int     `json:"meter" gorm:"column:meter;uniqueIndex:meters_meter_ts"`
	Timestamp int64   `json:"ts" gorm:"column:ts;uniqueIndex:meters_meter_ts"` // start of 15min slot
	Entity    entity  `json:"-" gorm:"foreignkey:Meter;references:Id"`
	Import    float64 `json:"import" gorm:"column:import"`
	Export    float64 `json:"export" gorm:"column:export"`
}

type entity struct {
	Id    int    `gorm:"column:id;primarykey"`
	Group string `gorm:"column:group;uniqueIndex:entities_group_name"`
	Name  string `gorm:"column:name;uniqueIndex:entities_group_name"`
}

func init() {
	db.Register(func(_ *gorm.DB) error {
		return SetupSchema()
	})
}

// SetupSchema is used for testing
func SetupSchema() error {
	m := db.Instance.Migrator()

	// entites: create entity first to make sure foreign keys for existing data work
	if err := db.Instance.AutoMigrate(new(entity)); err != nil {
		return err
	}

	// ensure home entity exists (reserves id=1 for legacy meter FK references)
	if _, err := createEntity(Home, Home); err != nil {
		return err
	}

	// enable FK constraints only here to make sure entity for metric exists
	if err := db.Instance.Exec("pragma foreign_keys(1)").Error; err != nil {
		return err
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

	rename := func(from, to string) error {
		if table := new(meter); m.HasColumn(table, from) && !m.HasColumn(table, to) {
			return m.RenameColumn(table, from, to)
		}
		return nil
	}

	// meter: split energy direction
	if err := rename("val", "import"); err != nil {
		return err
	}

	// meter: split energy direction #2
	if err := rename("pos", "import"); err != nil {
		return err
	}
	if err := rename("neg", "export"); err != nil {
		return err
	}

	// meter: ts migration
	if m.HasTable(new(meter)) {
		types, err := m.ColumnTypes(new(meter))
		if err != nil {
			return err
		}
		tsIdx := slices.IndexFunc(types, func(typ gorm.ColumnType) bool {
			return typ.Name() == "ts"
		})
		if tsIdx == -1 {
			return errors.New("missing meters.ts")
		}

		if tsTyp, _ := types[tsIdx].ColumnType(); !strings.EqualFold(tsTyp, "INTEGER") {
			db, err := db.Instance.DB()
			if err != nil {
				return err
			}

			if _, err := db.Exec(`UPDATE meters SET ts = unixepoch(ts)`); err != nil {
				return err
			}
		}
	}

	return db.Instance.AutoMigrate(new(meter))
}

// persist stores 15min consumption in kWh
func persist(entity entity, ts time.Time, imp, exp float64) error {
	return db.Instance.Create(&meter{
		Meter:     entity.Id,
		Timestamp: ts.Truncate(tariff.SlotDuration).Unix(),
		Import:    imp,
		Export:    exp,
	}).Error
}
