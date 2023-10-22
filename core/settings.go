package core

import (
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
)

// type Settings interface {
// 	SetString(val string)
// 	SetInt(val int64)
// 	SetFloat(val float64)
// 	SetTime(val time.Time)
// 	SetJson(val any) error
// 	SetBool(val bool)
// 	String(string, error)
// 	Int(int64, error)
// 	Float(float64, error)
// 	Time(time.Time, error)
// 	Bool(bool, error)
// 	Json(res any) error
// }

type Settings struct {
	Key string
}

func (s *Settings) SetString(key string, val string) {
	if s == nil {
		return
	}
	settings.SetString(s.Key+key, val)
}

func (s *Settings) SetInt(key string, val int64) {
	if s == nil {
		return
	}
	settings.SetInt(s.Key+key, val)
}

func (s *Settings) SetFloat(key string, val float64) {
	if s == nil {
		return
	}
	settings.SetFloat(s.Key+key, val)
}

func (s *Settings) SetTime(key string, val time.Time) {
	if s == nil {
		return
	}
	settings.SetTime(s.Key+key, val)
}

func (s *Settings) SetBool(key string, val bool) {
	if s == nil {
		return
	}
	settings.SetBool(s.Key+key, val)
}

// func (s *Settings) SetJson(key string, val any) error {
// 	if s == nil {
// 		return nil
// 	}
// 	return settings.SetJson(s.Key+key, val)
// }

func (s *Settings) String(key string) (string, error) {
	if s == nil {
		return "", nil
	}
	return settings.String(s.Key + key)
}

func (s *Settings) Int(key string) (int64, error) {
	if s == nil {
		return 0, nil
	}
	return settings.Int(s.Key + key)
}

func (s *Settings) Float(key string) (float64, error) {
	if s == nil {
		return 0, nil
	}
	return settings.Float(s.Key + key)
}

func (s *Settings) Time(key string) (time.Time, error) {
	if s == nil {
		return time.Time{}, nil
	}
	return settings.Time(s.Key + key)
}

func (s *Settings) Bool(key string) (bool, error) {
	if s == nil {
		return false, nil
	}
	return settings.Bool(s.Key + key)
}

// func (s *Settings) Json(key string, res any) error {
// 	return settings.Json(s.Key+key, &res)
// }
