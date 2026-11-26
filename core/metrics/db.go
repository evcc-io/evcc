package metrics

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

type meter struct {
	Meter        int      `json:"meter" gorm:"column:meter;primaryKey;autoIncrement:false"`
	Timestamp    int64    `json:"ts" gorm:"column:ts;primaryKey;autoIncrement:false"` // start of 15min slot
	Entity       entity   `json:"-" gorm:"foreignkey:Meter;references:Id"`
	Energy       float64  `json:"energy" gorm:"column:energy;type:numeric"` // numeric stores whole numbers compactly, real does not WITHOUT ROWID
	ReturnEnergy float64  `json:"returnEnergy" gorm:"column:return_energy;type:numeric"`
	SocTemp      *float64 `json:"socTemp,omitempty" gorm:"column:soc_temp;type:numeric"`    // at start of slot
	Recovered    int      `json:"recovered,omitempty" gorm:"column:recovered;type:numeric"` // slot origin, slotMeasured etc
}

type entity struct {
	Id                int      `gorm:"column:id;primarykey"`
	Group             string   `gorm:"column:group;uniqueIndex:entities_group_name"`
	Name              string   `gorm:"column:name;uniqueIndex:entities_group_name"`
	Title             string   `gorm:"column:title"`
	IsTemp            bool     `gorm:"column:is_temp"`             // soc_temp holds temperature, not soc
	EnergyMeter       *float64 `gorm:"column:energy_meter"`        // kWh, at last persisted slot
	ReturnEnergyMeter *float64 `gorm:"column:return_energy_meter"` // kWh, at last persisted slot
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
	if _, err := createEntity(Home, Home, Home); err != nil {
		return err
	}

	// grid: migrate old entities (title was the device ref) to the fixed title
	if err := db.Instance.Model(new(entity)).Where(`"group" = ? AND title <> ?`, Grid, Grid).Update("title", Grid).Error; err != nil {
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

	// meter: rename to energy/return_energy
	if err := rename("import", "energy"); err != nil {
		return err
	}
	if err := rename("export", "return_energy"); err != nil {
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

	if err := migrateMeterWithoutRowid(); err != nil {
		return err
	}

	if err := db.Instance.AutoMigrate(new(energySum), new(flowDay)); err != nil {
		return err
	}

	if err := backfill(); err != nil {
		return err
	}

	return loadDaily()
}

const withoutRowid = "WITHOUT ROWID"

// migrateMeterWithoutRowid keeps meters clustered on its primary key (meter, ts). A table from
// before stores every row twice, in the table and in its unique index, and is rebuilt once.
func migrateMeterWithoutRowid() error {
	m := db.Instance.Set("gorm:table_options", withoutRowid).Migrator()
	if !m.HasTable(new(meter)) {
		return m.CreateTable(new(meter))
	}

	current, err := meterTableCurrent()
	if err != nil {
		return err
	}
	if current {
		// an older version run in between recreates the unique index
		return db.Instance.Exec(`DROP INDEX IF EXISTS meters_meter_ts`).Error
	}

	// add the columns an older table lacks, so the copy finds them all. AutoMigrate would
	// also alter the changed column types, rebuilding the table once per column.
	stmt := &gorm.Statement{DB: db.Instance}
	if err := stmt.Parse(new(meter)); err != nil {
		return err
	}
	for _, name := range stmt.Schema.DBNames {
		if !m.HasColumn(new(meter), name) {
			if err := m.AddColumn(new(meter), name); err != nil {
				return err
			}
		}
	}

	if err := db.Instance.Transaction(func(tx *gorm.DB) error {
		// the unique index moves along and guarantees (meter, ts) holds no duplicates for the new primary key
		if err := tx.Exec(`ALTER TABLE meters RENAME TO meters_rowid`).Error; err != nil {
			return err
		}

		m := tx.Set("gorm:table_options", withoutRowid).Migrator()
		if err := m.CreateTable(new(meter)); err != nil {
			return err
		}

		types, err := m.ColumnTypes(new(meter))
		if err != nil {
			return err
		}
		cols := make([]string, 0, len(types))
		for _, typ := range types {
			cols = append(cols, `"`+typ.Name()+`"`)
		}
		list := strings.Join(cols, ",")

		// copy in key order along the old index, so the new table fills its pages in sequence. A row
		// without meter or ts no query can reach, and the primary key rejects it
		for _, stmt := range []string{
			`INSERT INTO meters (` + list + `) SELECT ` + list + ` FROM meters_rowid WHERE meter IS NOT NULL AND ts IS NOT NULL ORDER BY meter, ts`,
			`DROP TABLE meters_rowid`,
		} {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// the dropped table leaves its pages on the freelist, about half the file. Return them, which
	// also applies auto_vacuum to a database created before it was set. The rebuild is committed,
	// so a failure (e.g. not enough disk space for the copy) only costs space
	if err := db.Instance.Exec(`VACUUM`).Error; err != nil {
		log.WARN.Printf("vacuum after meters migration: %v", err)
	}
	return nil
}

// meterTableCurrent reports whether meters is WITHOUT ROWID with numeric measurement columns
func meterTableCurrent() (bool, error) {
	var ddl string
	if err := db.Instance.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'meters'`).Scan(&ddl).Error; err != nil {
		return false, err
	}
	if !strings.Contains(strings.ToUpper(ddl), withoutRowid) {
		return false, nil
	}

	types, err := db.Instance.Migrator().ColumnTypes(new(meter))
	if err != nil {
		return false, err
	}
	for _, typ := range types {
		switch typ.Name() {
		case "energy", "return_energy", "soc_temp":
			if !strings.EqualFold(typ.DatabaseTypeName(), "numeric") {
				return false, nil
			}
		}
	}
	return true, nil
}

// OnPersist, if set, is called with the slot start after a slot is written.
var OnPersist func(slot time.Time)

// persist stores a completed 15min slot
func persist(entity entity, ts time.Time, energy, returnEnergy float64, socTemp *float64, recovered bool) error {
	slot := ts.Truncate(tariff.SlotDuration)
	origin := slotMeasured
	if recovered {
		origin = slotRecovered
	}
	if err := writeDays(slot, slot.Add(tariff.SlotDuration), func(tx *gorm.DB) error {
		return tx.Create(&meter{
			Meter:        entity.Id,
			Timestamp:    slot.Unix(),
			Energy:       energy,
			ReturnEnergy: returnEnergy,
			SocTemp:      socTemp,
			Recovered:    origin,
		}).Error
	}); err != nil {
		return err
	}
	if OnPersist != nil {
		OnPersist(slot)
	}
	return nil
}
