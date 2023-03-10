package settings

import "time"

type Settings interface {
	Setter
	Getter
}

type Setter interface {
	SetString(key string, val string)
	SetInt(key string, val int64)
	SetFloat(key string, val float64)
	SetTime(key string, val time.Time)
	SetBool(key string, val bool)
	SetJson(key string, val any) error
}

type Getter interface {
	String(key string) (string, error)
	Int(key string) (int64, error)
	Float(key string) (float64, error)
	Time(key string) (time.Time, error)
	Bool(key string) (bool, error)
	Json(key string, res any) error
}
