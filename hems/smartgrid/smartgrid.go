package smartgrid

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
	"gorm.io/gorm"
)

func init() {
	db.Register(func(db *gorm.DB) error {
		return db.AutoMigrate(new(GridSession))
	})
}

func StartManage(typ Type, grid *float64, limit float64) (uint, error) {
	gs := GridSession{
		Created:    time.Now(),
		Type:       typ,
		GridPower:  grid,
		LimitPower: limit,
	}
	tx := db.Instance.Save(&gs)
	return gs.ID, tx.Error
}

func StopManage(id uint) error {
	return db.Instance.Where(&GridSession{
		ID: id,
	}).Updates(&GridSession{
		Finished: time.Now(),
	}).Error
}
