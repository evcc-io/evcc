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
	filePath string // Store the actual SQLite file path
)

func FilePath() string {
	return filePath
}

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
		filePath = file

		// TODO WAL mode "journal_mode(WAL)", "synchronous(NORMAL)"
		for _, pragma := range []string{"foreign_keys(1)", "auto_vacuum(INCREMENTAL)"} {
			// Add busy_timeout pragma if not already present
			if short, _, _ := strings.Cut(pragma, "("); strings.Contains(params, "_pragma="+short) {
				continue
			}

			// Append '&' if there are existing connection parameters
			if len(params) > 0 {
				params += "&"
			}

			// Add busy_timeout pragma to connection parameters
			params += "_pragma=" + pragma
		}

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

type backuper interface {
	NewBackup(string) (*sqlite3.Backup, error)
	NewRestore(string) (*sqlite3.Backup, error)
}

func runWithBackuper(ctx context.Context, fun func(backuper) (*sqlite3.Backup, error)) error {
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
		conn, ok := driverConn.(backuper)
		if !ok {
			return errors.New("invalid db type")
		}

		bck, err := fun(conn)
		if err != nil {
			return err
		}

		if _, err := bck.Step(-1); err != nil {
			return err
		}

		return bck.Finish()
	})
}

func Backup(ctx context.Context, target string) error {
	return runWithBackuper(ctx, func(conn backuper) (*sqlite3.Backup, error) {
		return conn.NewBackup(target)
	})
}

func Restore(ctx context.Context, target string) error {
	return runWithBackuper(ctx, func(conn backuper) (*sqlite3.Backup, error) {
		return conn.NewRestore(target)
	})
}
