package db

import (
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
