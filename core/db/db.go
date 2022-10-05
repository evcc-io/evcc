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
	Txn(startEnergy float64) *Transaction
	Persist(txn interface{})
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

// Txn creates a charging transaction
func (s *DB) Txn(meter float64) *Transaction {
	t := Transaction{
		Loadpoint:  s.name,
		Created:    time.Now(),
		MeterStart: meter,
	}

	return &t
}

// Persist creates or updates a transaction in the database
func (s *DB) Persist(txn interface{}) {
	s.log.TRACE.Printf("store: %+v", txn)

	if err := s.db.Save(txn).Error; err != nil {
		s.log.ERROR.Printf("store: %v", err)
	}
}
