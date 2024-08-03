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
	if s == nil {
		return
	}
	db.SetString(s.Key+key, val)
}

func (s *dbSettings) SetInt(key string, val int64) {
	if s == nil {
		return
	}
	db.SetInt(s.Key+key, val)
}

func (s *dbSettings) SetFloat(key string, val float64) {
	if s == nil {
		return
	}
	db.SetFloat(s.Key+key, val)
}

func (s *dbSettings) SetTime(key string, val time.Time) {
	if s == nil {
		return
	}
	db.SetTime(s.Key+key, val)
}

func (s *dbSettings) SetBool(key string, val bool) {
	if s == nil {
		return
	}
	db.SetBool(s.Key+key, val)
}

func (s *dbSettings) SetJson(key string, val any) error {
	if s == nil {
		return nil
	}
	return db.SetJson(s.Key+key, val)
}

func (s *dbSettings) String(key string) (string, error) {
	if s == nil {
		return "", nil
	}
	return db.String(s.Key + key)
}

func (s *dbSettings) Int(key string) (int64, error) {
	if s == nil {
		return 0, nil
	}
	return db.Int(s.Key + key)
}

func (s *dbSettings) Float(key string) (float64, error) {
	if s == nil {
		return 0, nil
	}
	return db.Float(s.Key + key)
}

func (s *dbSettings) Time(key string) (time.Time, error) {
	if s == nil {
		return time.Time{}, nil
	}
	return db.Time(s.Key + key)
}

func (s *dbSettings) Bool(key string) (bool, error) {
	if s == nil {
		return false, nil
	}
	return db.Bool(s.Key + key)
}

func (s *dbSettings) Json(key string, res any) error {
	if s == nil {
		return nil
	}
	return db.Json(s.Key+key, &res)
}
