package server

import (
	"strconv"
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

func intSetter(set func(int) error) func(string) error {
	return setterFunc(strconv.Atoi, set)
}
