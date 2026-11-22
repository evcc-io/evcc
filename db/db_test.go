package db

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func TestUnitNewDriver(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name             string
		driver           string
		dsn              string
		expectedFilePath string
		wantErr          bool
	}{
		{
			name:             "SQLite In-Memory",
			driver:           "sqlite",
			dsn:              ":memory:",
			expectedFilePath: ":memory:",
			wantErr:          false,
		},
		{
			name:             "SQLite File",
			driver:           "sqlite",
			dsn:              tmpDir + "/evcc.db",
			expectedFilePath: tmpDir + "/evcc.db",
			wantErr:          false,
		},
		{
			name:             "SQLite with connection parameters",
			driver:           "sqlite",
			dsn:              tmpDir + "evcc.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
			wantErr:          false,
			expectedFilePath: tmpDir + "evcc.db",
		},
		{
			name:    "Unsupported Driver",
			driver:  "postgresql",
			dsn:     "/var/lib/evcc/evcc.db",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset file path
			filePath = ""

			driver, err := New(test.driver, test.dsn)
			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, driver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, driver)
			}

			assert.Equal(t, test.expectedFilePath, FilePath())
		})
	}
}

// TestUnitWAL verifies WAL mode and that backup and restore leave no sidecar files behind
func TestUnitWAL(t *testing.T) {
	dir := t.TempDir()

	db, err := New("sqlite", dir+"/evcc.db")
	require.NoError(t, err)
	Instance = db

	var mode string
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&mode).Error)
	assert.Equal(t, "wal", mode)

	require.NoError(t, db.Exec("CREATE TABLE t (v integer)").Error)
	require.NoError(t, db.Exec("INSERT INTO t VALUES (1)").Error)

	backupDir := t.TempDir()
	require.NoError(t, Backup(t.Context(), backupDir+"/backup.db"))
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "backup must be a single self-contained file")

	require.NoError(t, db.Exec("INSERT INTO t VALUES (2)").Error)
	require.NoError(t, Restore(t.Context(), backupDir+"/backup.db"))

	var count int
	require.NoError(t, db.Raw("SELECT count(*) FROM t").Scan(&count).Error)
	assert.Equal(t, 1, count)

	// closing checkpoints the wal into the database file
	require.NoError(t, Close())
	entries, err = os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// shutdown closes again after restore and reset have closed
	require.NoError(t, Close())
}

// TestUnitConcurrentWrites verifies that the connection pool serves concurrent
// read-modify-write transactions and reads without SQLITE_BUSY
func TestUnitConcurrentWrites(t *testing.T) {
	db, err := New("sqlite", t.TempDir()+"/evcc.db")
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE t (id integer primary key, v integer)").Error)
	require.NoError(t, db.Exec("INSERT INTO t VALUES (1, 0)").Error)

	const workers, rounds = 8, 50

	var eg errgroup.Group
	for range workers {
		eg.Go(func() error {
			for range rounds {
				if err := db.Transaction(func(tx *gorm.DB) error {
					var v int
					if err := tx.Raw("SELECT v FROM t WHERE id = 1").Scan(&v).Error; err != nil {
						return err
					}
					return tx.Exec("UPDATE t SET v = ? WHERE id = 1", v+1).Error
				}); err != nil {
					return err
				}
			}
			return nil
		})
		eg.Go(func() error {
			for range rounds {
				var v int
				if err := db.Raw("SELECT v FROM t WHERE id = 1").Scan(&v).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
	require.NoError(t, eg.Wait())

	var v int
	require.NoError(t, db.Raw("SELECT v FROM t WHERE id = 1").Scan(&v).Error)
	assert.Equal(t, workers*rounds, v)
}

type migrationParent struct {
	Id int `gorm:"column:id;primarykey"`
}

func (migrationParent) TableName() string { return "migration_parents" }

type migrationChild struct {
	ParentId int             `gorm:"column:parent_id"`
	Parent   migrationParent `gorm:"foreignkey:ParentId;references:Id"`
}

func (migrationChild) TableName() string { return "migration_children" }

// TestUnitMigrateConstraint guards against migrator implementations that pin a
// connection and then query the pool again: the single connection deadlocks.
// In-memory databases keep a single connection.
func TestUnitMigrateConstraint(t *testing.T) {
	db, err := New("sqlite", ":memory:")
	require.NoError(t, err)

	// existing table without the foreign key, forces the migrator to recreate it
	require.NoError(t, db.Exec("CREATE TABLE migration_children (parent_id integer)").Error)

	done := make(chan error, 1)
	go func() { done <- db.AutoMigrate(new(migrationParent), new(migrationChild)) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("AutoMigrate deadlocked")
	}
}
