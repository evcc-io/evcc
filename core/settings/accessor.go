package settings

import (
	"time"

	"github.com/spf13/cast"
)

// accessor implements the typed scalar accessors on top of an untyped store.
// Json handling is left to the implementations as their storage formats differ.
type accessor struct {
	get func(key string) (any, error)
	set func(key string, val any)
}

func (s accessor) SetString(key string, val string) {
	s.set(key, val)
}

func (s accessor) SetInt(key string, val int64) {
	s.set(key, val)
}

func (s accessor) SetFloat(key string, val float64) {
	s.set(key, val)
}

func (s accessor) SetFloatPtr(key string, val *float64) {
	s.set(key, val)
}

func (s accessor) SetTime(key string, val time.Time) {
	s.set(key, val)
}

func (s accessor) SetBool(key string, val bool) {
	s.set(key, val)
}

func (s accessor) String(key string) (string, error) {
	val, err := s.get(key)
	if err != nil {
		return "", err
	}
	return cast.ToStringE(val)
}

func (s accessor) Int(key string) (int64, error) {
	val, err := s.get(key)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64E(val)
}

func (s accessor) Float(key string) (float64, error) {
	val, err := s.get(key)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(val)
}

func (s accessor) Time(key string) (time.Time, error) {
	val, err := s.get(key)
	if err != nil {
		return time.Time{}, err
	}
	return cast.ToTimeE(val)
}

func (s accessor) Bool(key string) (bool, error) {
	val, err := s.get(key)
	if err != nil {
		return false, err
	}
	return cast.ToBoolE(val)
}
