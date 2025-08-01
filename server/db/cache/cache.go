package cache

import (
	"encoding/json"
	"errors"

	"github.com/evcc-io/evcc/server/db"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("cache entry not found")

type cache struct {
	Key   string `json:"key" gorm:"primarykey"`
	Value string `json:"value"`
}

func Init() error {
	return db.Instance.AutoMigrate(new(cache))
}

func Put(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return db.Instance.Save(&cache{
		Key:   key,
		Value: string(data),
	}).Error
}

func Get(key string, result interface{}) error {
	var cacheEntry cache

	err := db.Instance.First(&cacheEntry, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	return json.Unmarshal([]byte(cacheEntry.Value), result)
}
