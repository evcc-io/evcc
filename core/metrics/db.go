package metrics

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meters_meter_ts"`
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meters_meter_ts"` // start of 15min slot
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

// Slot represents an aggregated energy time slot
type Slot struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Import float64   `json:"import"`
	Export float64   `json:"export"`
}

// Series represents a named series of energy slots
type Series struct {
	Name string `json:"name"`
	Data []Slot `json:"data"`
}

var aggregateFormats = map[string]string{
	"15m":   "%Y-%m-%d %H:%M",
	"hour":  "%Y-%m-%d %H:00",
	"day":   "%Y-%m-%d",
	"month": "%Y-%m",
}

var aggregateGoFormats = map[string]string{
	"15m":   "2006-01-02 15:04",
	"hour":  "2006-01-02 15:00",
	"day":   "2006-01-02",
	"month": "2006-01",
}

var aggregateDurations = map[string]func(time.Time) time.Time{
	"15m":   func(t time.Time) time.Time { return t.Add(15 * time.Minute) },
	"hour":  func(t time.Time) time.Time { return t.Add(time.Hour) },
	"day":   func(t time.Time) time.Time { return t.AddDate(0, 0, 1) },
	"month": func(t time.Time) time.Time { return t.AddDate(0, 1, 0) },
}

// QueryImportEnergy returns aggregated import energy data from the meters table
func QueryImportEnergy(from, to time.Time, aggregate string) ([]Series, error) {
	format, ok := aggregateFormats[aggregate]
	if !ok {
		return nil, errors.New("invalid aggregate value")
	}

	addDuration := aggregateDurations[aggregate]

	// use Go's tz offset instead of SQLite's 'localtime'
	tz := time.Now().Format("-07:00")

	tx := db.Instance.Table("meters m").
		Select(`e.name AS label, strftime('` + format + `', m.ts, '` + tz + `') AS bucket,
		COALESCE(SUM(m."import"), 0) AS import, COALESCE(SUM(m.export), 0) AS export`).
		Joins("JOIN entities e ON m.meter = e.id").
		Group("label, bucket").
		Order("label, bucket")

	if !from.IsZero() {
		tx = tx.Where("m.ts >= ?", from)
	}
	if !to.IsZero() {
		tx = tx.Where("m.ts < ?", to)
	}

	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := make(map[string][]Slot)
	var order []string

	for rows.Next() {
		var label, bucket string
		var imp, exp float64

		if err := rows.Scan(&label, &bucket, &imp, &exp); err != nil {
			return nil, err
		}

		start, err := time.ParseInLocation(aggregateGoFormats[aggregate], bucket, time.Now().Location())
		if err != nil {
			return nil, err
		}

		if _, exists := seriesMap[label]; !exists {
			order = append(order, label)
		}

		seriesMap[label] = append(seriesMap[label], Slot{
			Start:  start,
			End:    addDuration(start),
			Import: imp,
			Export: exp,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	res := make([]Series, 0, len(order))
	for _, name := range order {
		res = append(res, Series{
			Name: name,
			Data: seriesMap[name],
		})
	}

	return res, nil
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
			res.Group = Home
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

	return db.Instance.AutoMigrate(new(meter))
}

// persist stores 15min consumption in kWh
func persist(entity entity, ts time.Time, imp, exp float64) error {
	return db.Instance.Create(&meter{
		Meter:     entity.Id,
		Timestamp: ts.Truncate(tariff.SlotDuration),
		Import:    imp,
		Export:    exp,
	}).Error
}

// importProfile returns a 15min average meter profile in Wh. The profile
// is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func importProfile(entity entity, from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// use Go's tz offset instead of SQLite's 'localtime'
	tz := from.Format("-07:00")

	rows, err := db.Query(`SELECT min(ts) AS ts, avg(import) AS import
		FROM meters
		WHERE meter = ? AND ts >= ?
		GROUP BY strftime("%H:%M", ts, '`+tz+`')
		ORDER BY strftime("%H:%M", ts, '`+tz+`') ASC`, entity.Id, from,
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
