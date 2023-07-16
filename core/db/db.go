package db

import (
	"time"

	serverdb "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
	"gorm.io/gorm"
)

// DB is a SQL database storage service
type DB struct {
	log  *util.Logger
	db   *gorm.DB
	name string
}

type Database interface {
	Session(startEnergy float64) *Session
	CreatePowerState(utcTime time.Time) *PowerState
	Persist(session interface{})
	AddPowerState(powerState interface{})
	GetPowerStatesForDay(year uint16, month uint16, day uint16, offset uint16) Power
}

// New creates a database storage driver
func NewSession(name string) (*DB, error) {
	db := serverdb.Instance

	// TODO deprecate
	var err error
	if table := "transactions"; db.Migrator().HasTable(table) {
		err = db.Migrator().RenameTable(table, new(Session))
	}
	if err == nil {
		err = db.AutoMigrate(new(Session))
	}

	sessiondb := &DB{
		log:  util.NewLogger("db"),
		db:   db,
		name: name,
	}

	return sessiondb, err
}

// New creates a database storage driver
func NewPowerState(name string) (*DB, error) {
	db := serverdb.Instance

	// TODO deprecate
	var err error
	if table := "transactions"; db.Migrator().HasTable(table) {
		err = db.Migrator().RenameTable(table, new(PowerState))
	}
	if err == nil {
		err = db.AutoMigrate(new(PowerState))
	}

	powerdb := &DB{
		log:  util.NewLogger("db"),
		db:   db,
		name: name,
	}

	return powerdb, err
}

// Session creates a charging session
func (s *DB) Session(meter float64) *Session {
	t := Session{
		Loadpoint: s.name,
	}

	if meter > 0 {
		t.MeterStart = &meter
	}

	return &t
}

// Create a PowerState
func (s *DB) CreatePowerState(utcTime time.Time) *PowerState {
	p := PowerState{
		ID:     uint(utcTime.UnixMilli()),
		Year:   uint16(utcTime.Year()),
		Month:  uint16(utcTime.Month()),
		Day:    uint16(utcTime.Day()),
		Hour:   uint16(utcTime.Hour()),
		Minute: uint16(utcTime.Minute()),
	}
	return &p
}

func (s *DB) GetPowerStatesForDay(year uint16, month uint16, day uint16, offset uint16) Power {
	var result Power
	var tx *gorm.DB
	if offset > 0 {
		timeUTC := time.Date(int(year), time.Month(month), int(day), 00, 00, 00, 00, time.UTC)
		startTime := timeUTC.Add(-time.Duration(2) * time.Hour)
		endTime := timeUTC.Add(time.Duration(24) * time.Hour).Add(-time.Duration(2+1) * time.Hour)
		tx = s.db.Where("(year = ? and month = ? and day = ? and hour >= ?) or (year = ? and month = ? and day = ? and hour <= ?)",
			startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(),
			endTime.Year(), endTime.Month(), endTime.Day(), endTime.Hour()).Find(&result)

	} else {
		tx = s.db.Where("year = ? and month = ? and day = ?", year, month, day).Find(&result)
	}
	if tx.Error != nil {
		s.log.ERROR.Printf("GetPowerStatesForDay: %v", tx.Error)
	}
	return result
}

// Persist creates or updates a transaction in the database
func (s *DB) Persist(session interface{}) {
	if err := s.db.Save(session).Error; err != nil {
		s.log.ERROR.Printf("persist: %v", err)
	}
}

// Create or Update a Powerstate
func (s *DB) AddPowerState(powerstate interface{}) {
	if err := s.db.Save(powerstate).Error; err != nil {
		s.log.ERROR.Printf("persist: %v", err)
	}
}

// Return sessions
// TODO make this part of server/db
func (s *DB) Sessions() (Sessions, error) {
	var res Sessions
	tx := s.db.Find(&res)
	return res, tx.Error
}
