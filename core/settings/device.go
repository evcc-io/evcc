package settings

import (
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cast"
)

var _ Settings = (*DeviceSettings[api.Vehicle])(nil)

type DeviceSettings[T any] struct {
	dev config.ConfigurableDevice[T]
}

func NewDeviceSettingsAdapter[T any](dev config.ConfigurableDevice[T]) *DeviceSettings[T] {
	return &DeviceSettings[T]{dev}
}

func (s *DeviceSettings[T]) Update(dev config.ConfigurableDevice[T]) {
	s.dev = dev
}

func (s *DeviceSettings[T]) get(key string) any {
	conf := s.dev.Config().Other
	return conf[key]
}

func (s *DeviceSettings[T]) set(key string, val any) {
	conf := s.dev.Config().Other
	conf[key] = val
	s.dev.Update(conf, s.dev.Instance())
}

func (s *DeviceSettings[T]) SetString(key string, val string) {
	if s == nil {
		return
	}
	s.set(key, val)
}

func (s *DeviceSettings[T]) SetInt(key string, val int64) {
	if s == nil {
		return
	}
	s.set(key, val)
}

func (s *DeviceSettings[T]) SetFloat(key string, val float64) {
	if s == nil {
		return
	}
	s.set(key, val)
}

func (s *DeviceSettings[T]) SetTime(key string, val time.Time) {
	if s == nil {
		return
	}
	s.set(key, val)
}

func (s *DeviceSettings[T]) SetBool(key string, val bool) {
	if s == nil {
		return
	}
	s.set(key, val)
}

func (s *DeviceSettings[T]) SetJson(key string, val any) error {
	if s == nil {
		return nil
	}
	s.set(key, val)
	return nil
}

func (s *DeviceSettings[T]) String(key string) (string, error) {
	if s == nil {
		return "", nil
	}
	return cast.ToStringE(s.get(key))
}

func (s *DeviceSettings[T]) Int(key string) (int64, error) {
	if s == nil {
		return 0, nil
	}
	return cast.ToInt64E(s.get(key))
}

func (s *DeviceSettings[T]) Float(key string) (float64, error) {
	if s == nil {
		return 0, nil
	}
	return cast.ToFloat64E(s.get(key))
}

func (s *DeviceSettings[T]) Time(key string) (time.Time, error) {
	if s == nil {
		return time.Time{}, nil
	}
	return cast.ToTimeE(s.get(key))
}

func (s *DeviceSettings[T]) Bool(key string) (bool, error) {
	if s == nil {
		return false, nil
	}
	return cast.ToBoolE(s.get(key))
}

func (s *DeviceSettings[T]) Json(key string, res any) error {
	str, err := s.String(key)
	if str == "" || err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), &res)
}
