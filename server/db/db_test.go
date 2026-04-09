package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			FilePath = ""

			driver, err := New(test.driver, test.dsn)
			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, driver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, driver)
			}

			assert.Equal(t, test.expectedFilePath, FilePath)
		})
	}
}
