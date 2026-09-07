package db

import (
	"sync"

	"gorm.io/gorm"
)

var (
	mu       sync.Mutex
	registry []func(db *gorm.DB) error
)

func Register(fun func(db *gorm.DB) error) {
	mu.Lock()
	defer mu.Unlock()
	registry = append(registry, fun)
}
