package server

import (
	"strconv"
)

type setter struct {
	topic string
	fun   func(string) error
}

func setterFuncErr[T any](conv func(string) (T, error), set func(T) error) func(string) error {
	return func(payload string) error {
		val, err := conv(payload)
		if err == nil {
			err = set(val)
		}
		return err
	}
}

func setterFunc[T any](conv func(string) (T, error), set func(T)) func(string) error {
	return setterFuncErr(conv, func(val T) error {
		set(val)
		return nil
	})
}

func floatSetterErr(set func(float64) error) func(string) error {
	return setterFuncErr(parseFloat, set)
}

func floatSetter(set func(float64)) func(string) error {
	return setterFunc(parseFloat, set)
}

func intSetterErr(set func(int) error) func(string) error {
	return setterFuncErr(strconv.Atoi, set)
}

func intSetter(set func(int)) func(string) error {
	return setterFunc(strconv.Atoi, set)
}
