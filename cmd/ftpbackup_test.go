package cmd

import (
	"testing"
	"time"

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
