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

	switch driver {
	case "sqlite":

		// Example DSNs:
		//"path/to/database.db"
		// "~/database.db",
		// "database.db?cache=shared&journal_mode=WAL"
		// ":memory:"

		// Split database path and connection parameters
		dbPath, connectionParams, _ := strings.Cut(dsn, "?")

		file, err := homedir.Expand(dbPath)
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return nil, err
		}

		// Store the expanded file path for later use
		FilePath = file

		// Add busy_timeout pragma if not already present
		if !strings.Contains(connectionParams, "_pragma=busy_timeout") {
			// Append '&' if there are existing connection parameters
			if len(connectionParams) > 0 {
				connectionParams += "&"
			}

			// Add busy_timeout pragma to connection parameters
			connectionParams += "_pragma=busy_timeout(5000)"
		}

		connectionStr := file + "?" + connectionParams

		util.NewLogger("main").INFO.Println("using sqlite database:", connectionStr)

		dialect = sqlite.Open(connectionStr)
	// case "postgres":
	// 	dialect = postgres.Open(dsn)
	// case "mysql":
	// 	dialect = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("invalid database type: %s not in [sqlite]", driver)
	}

	return gorm.Open(dialect, &gorm.Config{
		Logger: &Logger{util.NewLogger("db")},
	})
}

func NewInstance(driver, dsn string) error {
	inst, err := New(strings.ToLower(driver), dsn)
	if err != nil {
		return err
	}

	Instance = inst

	mu.Lock()
	defer mu.Unlock()

	for _, f := range registry {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func Close() error {
	db, err := Instance.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
