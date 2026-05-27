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

func UpdateSession(id *uint, typ Type, circuitPower, limit float64, active bool) error {
	// start session
	if active && *id == 0 {
		var power *float64
		if circuitPower > 0 {
			power = new(circuitPower)
		}

		sid, err := StartManage(typ, power, limit)
		if err != nil {
			return err
		}

		*id = sid
	}

	// stop session
	if !active && *id != 0 {
		if err := StopManage(*id); err != nil {
			return err
		}

		*id = 0
	}

	return nil
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
