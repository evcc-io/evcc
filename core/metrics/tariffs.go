package metrics

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
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
