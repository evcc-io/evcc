package cache

import (
	"cmp"
	"encoding/json"
	"errors"
	"slices"

	"github.com/evcc-io/evcc/server/db"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("cache entry not found")

type Cache struct {
	Key   string `json:"key" gorm:"primarykey"`
	Value string `json:"value"`
}

func Init() error {
	return db.Instance.AutoMigrate(new(Cache))
}

func Put(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return db.Instance.Save(&Cache{
		Key:   key,
		Value: string(data),
	}).Error
}

func Get(key string, result any) error {
	var cacheEntry Cache

	err := db.Instance.First(&cacheEntry, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	return json.Unmarshal([]byte(cacheEntry.Value), result)
}

func All() ([]Cache, error) {
	var entries []Cache

	err := db.Instance.Find(&entries).Error
	if err != nil {
		return nil, err
	}

	// Sort by key for consistent output
	slices.SortFunc(entries, func(i, j Cache) int {
		return cmp.Compare(i.Key, j.Key)
	})

	return entries, nil
}

func Clear() error {
	return db.Instance.Delete(&Cache{}, "1=1").Error
}
