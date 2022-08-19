package transaction

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DBStorage is a SQL database storage service
type DBStorage struct {
	log *util.Logger
	db  *gorm.DB
}

// NewDB creates a database storage driver
func NewDB(path string) (*DBStorage, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	log := util.NewLogger("db")

	storage := &DBStorage{
		log: log,
		db:  db,
	}

	return storage, nil
}

// Database returns the gorm database
func (s *DBStorage) Database() *gorm.DB {
	return s.db
}

// Persist creates or updates a transaction in the database
func (s *DBStorage) Persist(txn interface{}) error {
	s.log.TRACE.Printf("store: %+v", txn)

	err := s.db.Save(txn).Error
	if err != nil {
		s.log.ERROR.Printf("persist: %v", err)
	}

	return err
}
