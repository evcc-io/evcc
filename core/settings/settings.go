package settings

import "time"

type Settings interface {
	SetString(key string, val string)
	SetInt(key string, val int64)
	SetFloat(key string, val float64)
	SetTime(key string, val time.Time)
	SetJson(key string, val any) error
	SetBool(key string, val bool)
	String(key string) (string, error)
	Int(key string) (int64, error)
	Float(key string) (float64, error)
	Time(key string) (time.Time, error)
	Bool(key string) (bool, error)
	Json(key string, res any) error
}
