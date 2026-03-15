package metrics

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

const (
	MeterHousehold = 1 // meter ID for household base load (backward compatible with master)
	// Loadpoint meter IDs: lpID + 2 (to avoid conflict with household=1)
	// e.g., loadpoint 0 = meter 2, loadpoint 1 = meter 3, etc.
)

type meter struct {
	Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:meter_ts"`
	Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:meter_ts"`
	Value     float64   `json:"val" gorm:"column:val"`
}

var ErrIncomplete = errors.New("insufficient historical data for meter profile (need 24 hours)")

func init() {
	db.Register(func(db *gorm.DB) error {
		return db.AutoMigrate(new(meter))
	})
}

// Persist stores 15min consumption in Wh for household total
func Persist(ts time.Time, value float64) error {
	return db.Instance.Create(meter{
		Meter:     MeterHousehold,
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// PersistLoadpoint stores 15min consumption in Wh for a specific loadpoint
func PersistLoadpoint(lpID int, ts time.Time, value float64) error {
	return db.Instance.Create(meter{
		Meter:     lpID + 2, // offset by 2 to avoid conflict with household=1
		Timestamp: ts.Truncate(15 * time.Minute),
		Value:     value,
	}).Error
}

// Profile returns a 15min average meter profile in Wh for household total.
// Profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func Profile(from time.Time) (*[96]float64, error) {
	return profileQuery(MeterHousehold, from)
}

// LoadpointProfile returns a 15min average meter profile in Wh for a specific loadpoint.
// Profile is sorted by timestamp starting at 00:00. It is guaranteed to contain 96 15min values.
func LoadpointProfile(lpID int, from time.Time) (*[96]float64, error) {
	return profileQuery(lpID+2, from)
}

// profileQuery is the internal implementation for querying meter profiles
func profileQuery(meterID int, from time.Time) (*[96]float64, error) {
	db, err := db.Instance.DB()
	if err != nil {
		return nil, err
	}

	// Use 'localtime' in strftime to fix https://github.com/evcc-io/evcc/discussions/23759
	var rows *sql.Rows
	rows, err = db.Query(`SELECT min(ts) AS ts, avg(val) AS val
		FROM meters
		WHERE meter = ? AND ts >= ?
		GROUP BY strftime("%H:%M", ts, 'localtime')
		ORDER BY strftime("%H:%M", ts, 'localtime') ASC`, meterID, from,
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
		return nil, fmt.Errorf("%w: got %d slots, need 96 (24 hours of 15-minute intervals)", ErrIncomplete, len(res))
	}

	return (*[96]float64)(res), nil
}
