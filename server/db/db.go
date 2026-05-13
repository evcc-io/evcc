package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/libtnb/sqlite"
	"github.com/mitchellh/go-homedir"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
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
		dbPath, params, _ := strings.Cut(dsn, "?")

		file, err := homedir.Expand(dbPath)
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return nil, err
		}

		// Store the expanded file path for later use
		FilePath = file

		addParam := func(typ, param string) {
			// Add busy_timeout pragma if not already present
			if short, _, _ := strings.Cut(param, "("); strings.Contains(params, typ+"="+short) {
				return
			}

			// Append '&' if there are existing connection parameters
			if len(params) > 0 {
				params += "&"
			}

			// Add busy_timeout pragma to connection parameters
			params += typ + "=" + param
		}

		// TODO "foreign_keys(1)" is only set in metrics migrator to ensure home entity exists
		for _, pragma := range []string{"busy_timeout(5000)", "synchronous(NORMAL)"} {
			addParam("_pragma", pragma)
		}

		// https://github.com/libtnb/sqlite/issues/15
		addParam("_time_format", "sqlite")

		connectionStr := file + "?" + params

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
	db, err := New(strings.ToLower(driver), dsn)
	if err != nil {
		return err
	}

	Instance = db

	mu.Lock()
	defer mu.Unlock()

	for _, f := range registry {
		if err := f(db); err != nil {
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

func Backup(ctx context.Context, target string) error {
	live, err := Instance.DB()
	if err != nil {
		return err
	}

	conn, err := live.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		type backuper interface {
			NewBackup(string) (*sqlite3.Backup, error)
			NewRestore(string) (*sqlite3.Backup, error)
		}

		conn, ok := driverConn.(backuper)
		if !ok {
			return errors.New("invalid db type")
		}

		bck, err := conn.NewBackup(target)
		if err != nil {
			return err
		}

		if _, err := bck.Step(-1); err != nil {
			return err
		}

		return bck.Finish()
	})
}
