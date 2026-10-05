package db

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func TestUnitFileMounted(t *testing.T) {
	mountinfo := `22 1 8:1 / / rw,relatime - ext4 /dev/sda1 rw
36 22 8:1 /home/user/.evcc /root/.evcc rw,relatime - ext4 /dev/sda1 rw
37 22 8:1 /home/user/evcc.db /data/evcc.db rw,relatime - ext4 /dev/sda1 rw
38 22 8:1 /home/user/my\040evcc.db /data/my\040evcc.db rw,relatime - ext4 /dev/sda1 rw
`
	for _, tc := range []struct {
		file    string
		mounted bool
	}{
		{"/data/evcc.db", true},
		{"/data/my evcc.db", true},
		{"/root/.evcc/evcc.db", false}, // directory mount
		{"/root/.evcc", true},
	} {
		assert.Equal(t, tc.mounted, fileMounted(strings.NewReader(mountinfo), tc.file), tc.file)
	}
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
func TestUnitMigrateConstraint(t *testing.T) {
	db, err := New("sqlite", t.TempDir()+"/evcc.db")
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
