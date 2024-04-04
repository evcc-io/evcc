package settings

import (
	"cmp"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

var ErrNotFound = errors.New("not found")

// setting is a settings entry
type setting struct {
	Key   string `json:"key" gorm:"primarykey"`
	Value string `json:"value"`
}

var (
	mu       sync.RWMutex
	settings []setting
	dirty    int32
)

func Init() error {
	err := db.Instance.AutoMigrate(new(setting))
	if err == nil {
		err = db.Instance.Find(&settings).Error
	}
	return err
}

func Persist() error {
	dirty := atomic.CompareAndSwapInt32(&dirty, 1, 0)
	if !dirty || len(settings) == 0 {
		// avoid "empty slice found"
		return nil
	}
	return db.Instance.Save(settings).Error
}

func All() []setting {
	mu.RLock()
	defer mu.RUnlock()

	res := slices.Clone(settings)
	slices.SortFunc(res, func(i, j setting) int {
		return cmp.Compare(i.Key, j.Key)
	})

	return res
}

func SetString(key string, val string) {
	mu.Lock()
	defer mu.Unlock()

	idx := slices.IndexFunc(settings, func(s setting) bool {
		return s.Key == key
	})

	if idx < 0 {
		settings = append(settings, setting{key, val})
		atomic.StoreInt32(&dirty, 1)
	} else if settings[idx].Value != val {
		settings[idx].Value = val
		atomic.StoreInt32(&dirty, 1)
	}
}

func SetInt(key string, val int64) {
	SetString(key, strconv.FormatInt(val, 10))
}

func SetFloat(key string, val float64) {
	SetString(key, strconv.FormatFloat(val, 'f', -1, 64))
}

func SetTime(key string, val time.Time) {
	SetString(key, val.Format(time.RFC3339))
}

func SetBool(key string, val bool) {
	SetString(key, strconv.FormatBool(val))
}

func SetJson(key string, val any) error {
	b, err := json.Marshal(val)
	if err == nil {
		SetString(key, string(b))
	}
	return err
}

func String(key string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()

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

func Time(key string) (time.Time, error) {
	s, err := String(key)
	if err != nil {
		return time.Now(), err
	}
	return time.Parse(time.RFC3339, s)
}

func Bool(key string) (bool, error) {
	s, err := String(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(s)
}

func Json(key string, res any) error {
	s, err := String(key)
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}
	return err
}

// wrapping Settings into a struct for better decoupling
type Settings struct{}

func (s Settings) String(key string) (string, error) {
	return String(key)
}

func (s Settings) SetString(key string, value string) {
	SetString(key, value)
}
