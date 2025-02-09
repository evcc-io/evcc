package settings

import (
	"time"

	db "github.com/evcc-io/evcc/server/db/settings"
)

var _ Settings = (*dbSettings)(nil)

type dbSettings struct {
	Key string
}

func NewDatabaseSettingsAdapter(key string) Settings {
	return &dbSettings{key}
}

func (s *dbSettings) SetString(key string, val string) {
	db.SetString(s.Key+key, val)
}

func (s *dbSettings) SetInt(key string, val int64) {
	db.SetInt(s.Key+key, val)
}

func (s *dbSettings) SetFloat(key string, val float64) {
	db.SetFloat(s.Key+key, val)
}

func (s *dbSettings) SetTime(key string, val time.Time) {
	db.SetTime(s.Key+key, val)
}

func (s *dbSettings) SetBool(key string, val bool) {
	db.SetBool(s.Key+key, val)
}

func (s *dbSettings) SetJson(key string, val any) error {
	return db.SetJson(s.Key+key, val)
}

func (s *dbSettings) String(key string) (string, error) {
	return db.String(s.Key + key)
}

func (s *dbSettings) Int(key string) (int64, error) {
	return db.Int(s.Key + key)
}

func (s *dbSettings) Float(key string) (float64, error) {
	return db.Float(s.Key + key)
}

func (s *dbSettings) Time(key string) (time.Time, error) {
	return db.Time(s.Key + key)
}

func (s *dbSettings) Bool(key string) (bool, error) {
	return db.Bool(s.Key + key)
}

func (s *dbSettings) Json(key string, res any) error {
	return db.Json(s.Key+key, &res)
}
