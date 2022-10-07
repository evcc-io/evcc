package settings

import (
	"errors"
	"strconv"

	"github.com/evcc-io/evcc/server/db"
	"golang.org/x/exp/slices"
)

var ErrNotFound = errors.New("not found")

// setting is a settings entry
type setting struct {
	Key   string `json:"key" gorm:"primarykey"`
	Value string `json:"value"`
}

var settings []setting

func Init() error {
	err := db.Instance.AutoMigrate(new(setting))
	if err == nil {
		err = db.Instance.Find(&settings).Error
	}
	return err
}

func Persist() error {
	if len(settings) == 0 {
		// avoid "empty slice found"
		return nil
	}
	return db.Instance.Save(settings).Error
}

func SetString(key string, val string) {
	idx := slices.IndexFunc(settings, func(s setting) bool {
		return s.Key == key
	})

	if idx < 0 {
		settings = append(settings, setting{key, val})
	} else {
		settings[idx].Value = val
	}
}

func SetInt(key string, val int64) {
	SetString(key, strconv.FormatInt(val, 10))
}

func SetFloat(key string, val float64) {
	SetString(key, strconv.FormatFloat(val, 'f', -1, 64))
}

func String(key string) (string, error) {
	idx := slices.IndexFunc(settings, func(s setting) bool {
		return s.Key == key
	})
	if idx < 0 {
		return "", ErrNotFound
	}
	return settings[idx].Value, nil
}

func Int(key string) (int64, error) {
	s, err := String(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func Float(key string) (float64, error) {
	s, err := String(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}
