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
	Persist(session interface{})
}

// New creates a database storage driver
func New(name string) (*DB, error) {
	db := &DB{
		log:  util.NewLogger("db"),
		db:   serverdb.Instance,
		name: name,
	}

	return db, nil
}

// Session creates a charging session
func (s *DB) Session(meter float64) *Session {
	t := Session{
		Loadpoint:  s.name,
		Created:    time.Now(),
		MeterStart: meter,
	}

	return &t
}

// Persist creates or updates a transaction in the database
func (s *DB) Persist(session interface{}) {
	s.log.TRACE.Printf("store: %+v", session)

	if err := s.db.Save(session).Error; err != nil {
		s.log.ERROR.Printf("store: %v", err)
	}
}
