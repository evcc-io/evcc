package cmd

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/stretchr/testify/require"
)

func TestNextDailyRun(t *testing.T) {
	now := time.Date(2026, 4, 18, 8, 30, 0, 0, time.UTC)

	t.Run("same day future time", func(t *testing.T) {
		runAt, err := nextDailyRun(now, "09:45")
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 4, 18, 9, 45, 0, 0, time.UTC), runAt)
	})

	t.Run("next day when schedule already passed", func(t *testing.T) {
		runAt, err := nextDailyRun(now, "06:00")
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 4, 19, 6, 0, 0, 0, time.UTC), runAt)
	})
}

func TestNextDailyRunInvalidSchedule(t *testing.T) {
	_, err := nextDailyRun(time.Now(), "invalid")
	require.Error(t, err)
}

func TestCurrentFTPBackupConfig(t *testing.T) {
	settings.SetString(keys.FTPBackup, "")

	t.Run("applies defaults", func(t *testing.T) {
		settings.SetString(keys.FTPBackup, "")

		conf, err := currentFTPBackupConfig(globalconfig.FTPBackup{})
		require.NoError(t, err)
		require.Equal(t, 21, conf.Port)
		require.Equal(t, "03:00", conf.Schedule)
		require.Equal(t, "30s", conf.Timeout)
	})

	t.Run("loads updated settings", func(t *testing.T) {
		settings.SetString(keys.FTPBackup, "")

		require.NoError(t, settings.SetJson(keys.FTPBackup, globalconfig.FTPBackup{
			Host:     "example.org",
			Schedule: "09:15",
			Timeout:  "45s",
			Port:     2121,
		}))

		conf, err := currentFTPBackupConfig(globalconfig.FTPBackup{})
		require.NoError(t, err)
		require.Equal(t, "example.org", conf.Host)
		require.Equal(t, "09:15", conf.Schedule)
		require.Equal(t, "45s", conf.Timeout)
		require.Equal(t, 2121, conf.Port)
	})

	t.Run("rejects invalid timeout", func(t *testing.T) {
		settings.SetString(keys.FTPBackup, "")

		require.NoError(t, settings.SetJson(keys.FTPBackup, globalconfig.FTPBackup{
			Host:     "example.org",
			Schedule: "09:15",
			Timeout:  "invalid",
		}))

		_, err := currentFTPBackupConfig(globalconfig.FTPBackup{})
		require.Error(t, err)
	})
}
