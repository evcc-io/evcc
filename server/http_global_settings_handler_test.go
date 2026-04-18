package server

import (
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONSettingFTPBackup(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		err := validateJSONSetting(keys.FTPBackup, &globalconfig.FTPBackup{
			Schedule: "12:30",
			Timeout:  "30s",
		})
		require.NoError(t, err)
	})

	t.Run("invalid schedule", func(t *testing.T) {
		err := validateJSONSetting(keys.FTPBackup, &globalconfig.FTPBackup{
			Schedule: "25:30",
		})
		require.Error(t, err)
	})

	t.Run("invalid timeout", func(t *testing.T) {
		err := validateJSONSetting(keys.FTPBackup, &globalconfig.FTPBackup{
			Timeout: "-5s",
		})
		require.Error(t, err)
	})
}
