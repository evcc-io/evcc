package server

import (
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
)

type setter struct {
	topic string
	fun   func(string) error
}

func isEmpty(payload string) bool {
	return slices.Contains([]string{"-", "nil", "null", "none"}, payload)
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

// ptrSetter updates pointer api
func ptrSetter[T any](conv func(string) (T, error), set func(*T) error) func(string) error {
	return func(payload string) error {
		var val *T
		v, err := conv(payload)
		if err == nil {
			val = &v
		} else if !isEmpty(payload) {
			return err
		}
		return set(val)
	}
}

func floatSetter(set func(float64) error) func(string) error {
	return setterFunc(parseFloat, set)
}

func floatPtrSetter(set func(*float64) error) func(string) error {
	return ptrSetter(parseFloat, set)
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

func planStrategySetter(set func(api.PlanStrategy) error) func(string) error {
	return func(payload string) error {
		var res api.PlanStrategy
		if err := json.Unmarshal([]byte(payload), &res); err != nil {
			return err
		}

		return set(res)
	}
}

func fullChargeSetter(set func(api.FullCharge) error) func(string) error {
	return func(payload string) error {
		var res api.FullCharge
		if err := json.Unmarshal([]byte(payload), &res); err != nil {
			return err
		}

		return set(res)
	}
}

// fullChargeDoneSetter records a full charge at the RFC3339 time of the payload, now if empty
func fullChargeDoneSetter(v vehicle.API) func(string) error {
	return func(payload string) error {
		ts := time.Now()
		if payload != "" && payload != "now" {
			var err error
			if ts, err = time.Parse(time.RFC3339, payload); err != nil {
				return err
			}
		}

		return fullChargeDone(v, ts)
	}
}

func planGoalSetter[T any](set func(time.Time, T) error) func(string) error {
	return func(payload string) error {
		var plan planGoal[T]
		if err := json.Unmarshal([]byte(payload), &plan); err != nil {
			return err
		}

		return set(plan.Time, plan.Value)
	}
}
