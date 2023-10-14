package session

import (
	"github.com/evcc-io/evcc/util"
	"gorm.io/gorm"
)

// DB is a SQL database storage service
type DB struct {
	log  *util.Logger
	db   *gorm.DB
	name string
}

// NewStore creates a session store
func NewStore(name string, db *gorm.DB) (*DB, error) {
	err := db.AutoMigrate(new(Session))

	sessiondb := &DB{
		log:  util.NewLogger("db"),
		db:   db,
		name: name,
	}

	return sessiondb, err
}

// New creates a charging session
func (s *DB) New(meter float64) *Session {
	t := Session{
		Loadpoint: s.name,
	}

	if meter > 0 {
		t.MeterStart = &meter
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

func (s *DB) ClosePendingSessionsInHistory(chargeMeterTotal float64) error {
	var res Sessions
	if tx := s.db.Find(&res, map[string]interface{}{"finished": "0001-01-01 00:00:00+00:00", "Loadpoint": s.name}); tx.Error != nil {
		return tx.Error
	}

	for _, session := range res {
		var nextSession Session

		var tx *gorm.DB
		if tx = s.db.Limit(1).Order("ID").Find(&nextSession, "ID > ? AND Loadpoint = ?", session.ID, s.name); tx.Error != nil {
			return tx.Error
		}

		if tx.RowsAffected == 0 {
			// no successor, this is the most recent session and it is open
			session.MeterStop = &chargeMeterTotal
		} else {
			session.MeterStop = nextSession.MeterStart
		}

		session.ChargedEnergy = *session.MeterStop - *session.MeterStart
		s.Persist(session)
	}

	return nil
}
