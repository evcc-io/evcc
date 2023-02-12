package db

import (
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
	Persist(session interface{})
}

// New creates a database storage driver
func New(name string) (*DB, error) {
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

// Session creates a charging session
func (s *DB) Session(meter float64) *Session {
	t := Session{
		Loadpoint:  s.name,
		MeterStart: meter,
	}

	return &t
}

// Persist creates or updates a transaction in the database
func (s *DB) Persist(session interface{}) {
	if err := s.db.Save(session).Error; err != nil {
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
