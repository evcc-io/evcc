package metrics

import (
	"database/sql"
	"errors"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

const (
	MeterTypeHousehold = 1
	MeterTypeLoadpoint = 2
	
	// LoadpointNameHousehold is the identifier for household base load (non-loadpoint consumption)
	LoadpointNameHousehold = "_household"
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meter_ts"`
	Loadpoint string    `json:"loadpoint" gorm:"column:loadpoint;uniqueIndex:meter_ts;default:'_household'"` // loadpoint name: "_household" for base load, or loadpoint title for heaters
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meter_ts"`
	Value     float64   `json:"val" gorm:"column:val"`
}

var ErrIncomplete = errors.New("meter profile incomplete")

func init() {
	db.Register(func(db *gorm.DB) error {
		return db.AutoMigrate(new(meter))
	})
}

// Persist stores 15min consumption in Wh for household total
func Persist(ts time.Time, value float64) error {
	return db.Instance.Create(meter{
		Meter:     MeterTypeHousehold,
		Loadpoint: LoadpointNameHousehold,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// PersistLoadpoint stores 15min consumption in Wh for a specific loadpoint
func PersistLoadpoint(loadpointName string, ts time.Time, value float64) error {
	return db.Instance.Create(meter{
		Meter:     MeterTypeLoadpoint,
		Loadpoint: loadpointName,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// Profile returns a 15min average meter profile in Wh for household total.
// Profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func Profile(from time.Time) (*[96]float64, error) {
	return profileQuery(MeterTypeHousehold, LoadpointNameHousehold, from)
}

// LoadpointProfile returns a 15min average meter profile in Wh for a specific loadpoint.
// Profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func LoadpointProfile(loadpointName string, from time.Time) (*[96]float64, error) {
	return profileQuery(MeterTypeLoadpoint, loadpointName, from)
}

// profileQuery is the internal implementation for querying meter profiles
func profileQuery(meterType int, loadpointName string, from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// Use 'localtime' in strftime to fix https://github.com/evcc-io/evcc/discussions/23759
	var rows *sql.Rows
	rows, err = db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ? AND loadpoint = ? AND ts >= ?
		GROUP BY strftime("%H:%M", ts, 'localtime')
		ORDER BY strftime("%H:%M", ts, 'localtime') ASC`, meterType, loadpointName, from,
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
