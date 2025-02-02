package server

import (
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
)

type setter struct {
	topic string
	fun   func(string) error
}

func setterFunc[T any](conv func(string) (T, error), set func(T) error) func(string) error {
	return func(payload string) error {
		val, err := conv(payload)
		if err == nil {
			err = set(val)
		}
		return err
	}
}

func floatSetter(set func(float64) error) func(string) error {
	return setterFunc(parseFloat, set)
}

func floatPtrSetter(set func(*float64) error) func(string) error {
	return func(s string) error {
		var val *float64
		if f, err := parseFloat(s); err == nil {
			val = &f
		} else if s != "" {
			return err
		}
		return set(val)
	}
}

func intSetter(set func(int) error) func(string) error {
	return setterFunc(strconv.Atoi, set)
}

func boolSetter(set func(bool) error) func(string) error {
	return setterFunc(func(v string) (bool, error) {
		return cast.ToBoolE(v)
	}, set)
}

func durationSetter(set func(time.Duration) error) func(string) error {
	return setterFunc(util.ParseDuration, set)
}
