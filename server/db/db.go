package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func New(path string) (err error) {
	Instance, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	return
}
