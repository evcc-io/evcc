package globalconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFTPBackupValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		conf := FTPBackup{
			Schedule: "12:30",
			Timeout:  "30s",
		}
		require.NoError(t, conf.Validate())
	})

	t.Run("invalid schedule", func(t *testing.T) {
		conf := FTPBackup{
			Schedule: "25:30",
		}
		require.Error(t, conf.Validate())
	})

	t.Run("invalid timeout", func(t *testing.T) {
		conf := FTPBackup{
			Timeout: "-5s",
		}
		require.Error(t, conf.Validate())
	})
}
