package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func New(driver, path string) (*gorm.DB, error) {
	var dialect gorm.Dialector

	switch driver {
	case "sqlite":
		dialect = sqlite.Open(path)
	case "postgres":
		dialect = postgres.Open(path)
	case "mysql":
		dialect = mysql.Open(path)
	default:
		return nil, fmt.Errorf("database type %s not valid. Must be one of (sqlite, postgres, mysql)", driver)
	}
	return gorm.Open(dialect, &gorm.Config{})
}

func NewGlobal(driver, path string) (err error) {
	Instance, err = New(driver, path)
	return
}
