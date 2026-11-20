package metrics

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tariffValue struct {
	Timestamp   int64    `gorm:"column:ts;uniqueIndex"` // 15min boundary
	Grid        *float64 `gorm:"column:grid"`
	FeedIn      *float64 `gorm:"column:feedin"`
	Co2         *float64 `gorm:"column:co2"`
	Temperature *float64 `gorm:"column:temperature"`
}

func (tariffValue) TableName() string {
	return "tariffs"
}

func init() {
	db.Register(func(_ *gorm.DB) error {
		return db.Instance.AutoMigrate(new(tariffValue))
	})
}

// ErrInvalidUsage is returned for an unknown tariff usage
var ErrInvalidUsage = errors.New("invalid usage")

// tariffUsages are the deletable usages, named after their table column
var tariffUsages = []string{"grid", "feedin", "co2", "temperature"}

// DeleteTariffs removes the persisted values in [from,to). An empty usage drops
// the entire row, otherwise only that usage is cleared. Both bounds are
// required, a full wipe is /api/db/reset. The count is the number of affected
// rows; rows dropped by the cleanup are a subset of the cleared ones.
func DeleteTariffs(from, to time.Time, usage string) (int64, error) {
	if from.IsZero() || to.IsZero() {
		return 0, errors.New("missing from/to")
	}

	inRange := func() *gorm.DB {
		return db.Instance.Where("ts >= ? AND ts < ?", from.Unix(), to.Unix())
	}

	if usage == "" {
		res := inRange().Delete(new(tariffValue))
		return res.RowsAffected, res.Error
	}

	// guards the column interpolated below
	if !slices.Contains(tariffUsages, usage) {
		return 0, fmt.Errorf("%w: %s (valid: %s)", ErrInvalidUsage, usage, strings.Join(tariffUsages, ", "))
	}

	res := inRange().Model(new(tariffValue)).
		Where(usage+" IS NOT NULL").
		Update(usage, gorm.Expr("NULL"))
	if res.Error != nil {
		return 0, res.Error
	}

	// drop the rows that no longer hold any value
	err := inRange().
		Where("grid IS NULL AND feedin IS NULL AND co2 IS NULL AND temperature IS NULL").
		Delete(new(tariffValue)).Error

	return res.RowsAffected, err
}

// PersistTariffs stores the tariff values at the given 15min boundary, nil values omitted
func PersistTariffs(ts time.Time, grid, feedin, co2, temperature *float64) error {
	if grid == nil && feedin == nil && co2 == nil && temperature == nil {
		return nil
	}

	return db.Instance.Clauses(clause.OnConflict{DoNothing: true}).Create(&tariffValue{
		Timestamp:   ts.Unix(),
		Grid:        grid,
		FeedIn:      feedin,
		Co2:         co2,
		Temperature: temperature,
	}).Error
}

// TariffSlot is one persisted 15 minute slot of tariff values, or the average
// of a bucket with the slot range when aggregated
type TariffSlot struct {
	Timestamp   int64     `json:"-" gorm:"column:ts"`
	Start       time.Time `json:"start" gorm:"-"`
	Grid        *float64  `json:"grid,omitempty"`
	GridMin     *float64  `json:"gridMin,omitempty"`
	GridMax     *float64  `json:"gridMax,omitempty"`
	FeedIn      *float64  `json:"feedin,omitempty"`
	FeedInMin   *float64  `json:"feedinMin,omitempty"`
	FeedInMax   *float64  `json:"feedinMax,omitempty"`
	Co2         *float64  `json:"co2,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

// QueryTariffs returns the persisted slots in [from,to), zero bounds are open.
// With an aggregate of hour, day or month the slots are averaged per bucket.
func QueryTariffs(from, to time.Time, aggregate string) ([]TariffSlot, error) {
	tx := db.Instance.Table("tariffs").Order("ts")
	if !from.IsZero() {
		tx = tx.Where("ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("ts < ?", to.Unix())
	}

	if aggregate == "" || aggregate == "15m" {
		tx = tx.Select("ts, grid, feedin AS feed_in, co2, temperature")
	} else {
		format, ok := aggregateFormats[aggregate]
		if !ok {
			return nil, fmt.Errorf("invalid aggregate: %s", aggregate)
		}
		tx = tx.Select(`MIN(ts) AS ts,
			AVG(grid) AS grid, MIN(grid) AS grid_min, MAX(grid) AS grid_max,
			AVG(feedin) AS feed_in, MIN(feedin) AS feed_in_min, MAX(feedin) AS feed_in_max`).
			Group(fmt.Sprintf(`strftime('%s', ts, 'unixepoch', 'localtime')`, format))
	}

	var res []TariffSlot
	if err := tx.Scan(&res).Error; err != nil {
		return nil, err
	}
	for i := range res {
		res[i].Start = time.Unix(res[i].Timestamp, 0)
	}
	return res, nil
}
