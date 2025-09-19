package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/glebarez/sqlite"
	"github.com/mitchellh/go-homedir"
	"gorm.io/gorm"
)

var (
	Instance *gorm.DB
	FilePath string // Store the actual SQLite file path
)

func New(driver, dsn string) (*gorm.DB, error) {
	var dialect gorm.Dialector
	log := util.NewLogger("db")

	switch driver {
	case "sqlite":
		file, err := homedir.Expand(dsn)
		if err != nil {
			return nil, err
		}

		// Store the expanded file path for later use
		FilePath = file

		log.INFO.Println("using sqlite database:", file)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return nil, err
		}
		// avoid busy errors
		dialect = sqlite.Open(file + "?_pragma=busy_timeout(5000)")
	// case "postgres":
	// 	dialect = postgres.Open(dsn)
	// case "mysql":
	// 	dialect = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("invalid database type: %s not in [sqlite]", driver)
	}

	return gorm.Open(dialect, &gorm.Config{
		Logger: &Logger{log},
	})
}

func NewInstance(driver, dsn string) (err error) {
	Instance, err = New(strings.ToLower(driver), dsn)
	return
}

func Close() error {
	db, err := Instance.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
