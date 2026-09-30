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
