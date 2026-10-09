package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNextOccurrenceBasic(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	require.NoError(t, err)

	t.Run("target in future today", func(t *testing.T) {
		// Monday 06:00 UTC
		now := time.Date(2026, 8, 17, 6, 0, 0, 0, loc)
		weekdays := []int{1} // Monday
		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "UTC")
		require.NoError(t, err)
		expected := time.Date(2026, 8, 17, 7, 0, 0, 0, loc)
		assert.Equal(t, expected, target)
	})

	t.Run("target already passed today", func(t *testing.T) {
		// Monday 08:00 UTC -> should pick next Monday if only Monday configured
		now := time.Date(2026, 8, 17, 8, 0, 0, 0, loc)
		weekdays := []int{1} // Monday
		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "UTC")
		require.NoError(t, err)
		expected := time.Date(2026, 8, 24, 7, 0, 0, 0, loc)
		assert.Equal(t, expected, target)
	})

	t.Run("target already passed today with multiple weekdays", func(t *testing.T) {
		// Monday 08:00 UTC -> should pick Tuesday 07:00
		now := time.Date(2026, 8, 17, 8, 0, 0, 0, loc)
		weekdays := []int{1, 2, 3, 4, 5} // Mon-Fri
		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "UTC")
		require.NoError(t, err)
		expected := time.Date(2026, 8, 18, 7, 0, 0, 0, loc)
		assert.Equal(t, expected, target)
	})

	t.Run("target exactly now", func(t *testing.T) {
		// Monday 07:00 UTC
		now := time.Date(2026, 8, 17, 7, 0, 0, 0, loc)
		weekdays := []int{1}
		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "UTC")
		require.NoError(t, err)
		expected := time.Date(2026, 8, 17, 7, 0, 0, 0, loc)
		assert.Equal(t, expected, target)
	})

	t.Run("GetNextOccurrence default now wrapper", func(t *testing.T) {
		allWeekdays := []int{0, 1, 2, 3, 4, 5, 6}
		target, err := GetNextOccurrence(allWeekdays, "23:59", "UTC")
		require.NoError(t, err)
		assert.False(t, target.IsZero())
	})
}

func TestGetNextOccurrenceErrors(t *testing.T) {
	now := time.Date(2026, 8, 17, 6, 0, 0, 0, time.UTC)

	t.Run("invalid timezone", func(t *testing.T) {
		_, err := GetNextOccurrenceAt(now, []int{1}, "07:00", "NonExistent/Timezone")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("invalid time format", func(t *testing.T) {
		_, err := GetNextOccurrenceAt(now, []int{1}, "25:00", "UTC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time format")

		_, err = GetNextOccurrenceAt(now, []int{1}, "invalid", "UTC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time format")
	})

	t.Run("no matching weekday found", func(t *testing.T) {
		_, err := GetNextOccurrenceAt(now, []int{}, "07:00", "UTC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid weekday found")

		_, err = GetNextOccurrenceAt(now, []int{99}, "07:00", "UTC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid weekday found")
	})
}

func TestGetNextOccurrenceSpringDSTSwitchover(t *testing.T) {
	// Europe/Berlin spring switchover: Sunday, March 29, 2026
	// At 02:00 CET (+01:00), clocks turn forward to 03:00 CEST (+02:00).
	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	t.Run("resolve across switchover from Saturday night", func(t *testing.T) {
		// Saturday March 28, 2026 23:00 CET (UTC: 22:00Z)
		now := time.Date(2026, 3, 28, 23, 0, 0, 0, berlinLoc)
		weekdays := []int{0} // Sunday only

		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "Europe/Berlin")
		require.NoError(t, err)

		// Sunday March 29, 2026 07:00 CEST (UTC: 05:00Z)
		expected := time.Date(2026, 3, 29, 7, 0, 0, 0, berlinLoc)
		assert.Equal(t, expected, target)
		assert.Equal(t, time.Sunday, target.Weekday())

		// Difference across transition is 7 hours of physical duration (22:00 UTC to 05:00 UTC next day)
		duration := target.Sub(now)
		assert.Equal(t, 7*time.Hour, duration)

		// Wall-clock time should be exactly 07:00:00
		assert.Equal(t, 7, target.Hour())
		assert.Equal(t, 0, target.Minute())
		assert.Equal(t, 0, target.Second())
	})

	t.Run("skipped hour normalization during spring forward", func(t *testing.T) {
		// 02:30 does not exist on Sunday March 29, 2026 in Europe/Berlin
		now := time.Date(2026, 3, 28, 23, 0, 0, 0, berlinLoc)
		weekdays := []int{0} // Sunday

		target, err := GetNextOccurrenceAt(now, weekdays, "02:30", "Europe/Berlin")
		require.NoError(t, err)

		// Go standard library normalizes 02:30 CET forward to 03:30 CEST
		assert.Equal(t, time.Sunday, target.Weekday())
		assert.Equal(t, berlinLoc, target.Location())
		assert.False(t, target.IsZero())
	})

	t.Run("plan after spring switchover on same day", func(t *testing.T) {
		// Sunday March 29, 2026 04:00 CEST (after jump to CEST)
		now := time.Date(2026, 3, 29, 4, 0, 0, 0, berlinLoc)
		weekdays := []int{0, 1} // Sunday and Monday

		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "Europe/Berlin")
		require.NoError(t, err)

		// Same day Sunday 07:00 CEST
		expected := time.Date(2026, 3, 29, 7, 0, 0, 0, berlinLoc)
		assert.Equal(t, expected, target)
	})

	t.Run("resolve from Sunday afternoon to Monday morning", func(t *testing.T) {
		// Sunday March 29, 2026 12:00 CEST
		now := time.Date(2026, 3, 29, 12, 0, 0, 0, berlinLoc)
		weekdays := []int{1} // Monday only

		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "Europe/Berlin")
		require.NoError(t, err)

		// Monday March 30, 2026 07:00 CEST (UTC: 05:00Z)
		expected := time.Date(2026, 3, 30, 7, 0, 0, 0, berlinLoc)
		assert.Equal(t, expected, target)
		assert.Equal(t, time.Monday, target.Weekday())
		assert.Equal(t, 19*time.Hour, target.Sub(now))
	})
}

func TestGetNextOccurrenceAutumnDSTSwitchover(t *testing.T) {
	// Europe/Berlin autumn switchover: Sunday, October 25, 2026
	// At 03:00 CEST (+02:00), clocks turn back to 02:00 CET (+01:00).
	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	t.Run("resolve across switchover from Saturday night", func(t *testing.T) {
		// Saturday October 24, 2026 23:00 CEST (UTC: 21:00Z)
		now := time.Date(2026, 10, 24, 23, 0, 0, 0, berlinLoc)
		weekdays := []int{0} // Sunday only

		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "Europe/Berlin")
		require.NoError(t, err)

		// Sunday October 25, 2026 07:00 CET (UTC: 06:00Z)
		expected := time.Date(2026, 10, 25, 7, 0, 0, 0, berlinLoc)
		assert.Equal(t, expected, target)
		assert.Equal(t, time.Sunday, target.Weekday())

		// Difference across transition is 9 hours of physical duration (21:00 UTC to 06:00 UTC next day)
		duration := target.Sub(now)
		assert.Equal(t, 9*time.Hour, duration)

		// Wall-clock time should be exactly 07:00:00
		assert.Equal(t, 7, target.Hour())
		assert.Equal(t, 0, target.Minute())
		assert.Equal(t, 0, target.Second())
	})

	t.Run("repeated hour resolution during autumn fallback", func(t *testing.T) {
		// 02:30 occurs twice on Sunday October 25, 2026 in Europe/Berlin
		now := time.Date(2026, 10, 24, 23, 0, 0, 0, berlinLoc)
		weekdays := []int{0} // Sunday

		target, err := GetNextOccurrenceAt(now, weekdays, "02:30", "Europe/Berlin")
		require.NoError(t, err)

		assert.Equal(t, time.Sunday, target.Weekday())
		assert.Equal(t, berlinLoc, target.Location())
		assert.False(t, target.IsZero())
	})

	t.Run("resolve from Sunday autumn afternoon to Monday morning", func(t *testing.T) {
		// Sunday October 25, 2026 12:00 CET
		now := time.Date(2026, 10, 25, 12, 0, 0, 0, berlinLoc)
		weekdays := []int{1} // Monday only

		target, err := GetNextOccurrenceAt(now, weekdays, "07:00", "Europe/Berlin")
		require.NoError(t, err)

		// Monday October 26, 2026 07:00 CET (UTC: 06:00Z)
		expected := time.Date(2026, 10, 26, 7, 0, 0, 0, berlinLoc)
		assert.Equal(t, expected, target)
		assert.Equal(t, time.Monday, target.Weekday())
		assert.Equal(t, 19*time.Hour, target.Sub(now))
	})
}

func TestGetNextOccurrenceVariousTimezones(t *testing.T) {
	t.Run("America/New_York spring switchover", func(t *testing.T) {
		// US Spring DST: Sunday, March 8, 2026 (02:00 EST -> 03:00 EDT)
		nyLoc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		now := time.Date(2026, 3, 7, 23, 0, 0, 0, nyLoc)
		target, err := GetNextOccurrenceAt(now, []int{0}, "08:00", "America/New_York")
		require.NoError(t, err)

		expected := time.Date(2026, 3, 8, 8, 0, 0, 0, nyLoc)
		assert.Equal(t, expected, target)
		// 23:00 EST (04:00 UTC) to 08:00 EDT (12:00 UTC) = 8 physical hours
		assert.Equal(t, 8*time.Hour, target.Sub(now))
	})

	t.Run("Australia/Sydney southern hemisphere switchover", func(t *testing.T) {
		// Sydney Autumn fallback: Sunday, April 5, 2026 (03:00 AEDT -> 02:00 AEST)
		sydLoc, err := time.LoadLocation("Australia/Sydney")
		require.NoError(t, err)

		now := time.Date(2026, 4, 4, 22, 0, 0, 0, sydLoc)
		target, err := GetNextOccurrenceAt(now, []int{0}, "08:00", "Australia/Sydney")
		require.NoError(t, err)

		expected := time.Date(2026, 4, 5, 8, 0, 0, 0, sydLoc)
		assert.Equal(t, expected, target)
		// 22:00 AEDT (11:00 UTC) to 08:00 AEST (22:00 UTC) = 11 physical hours
		assert.Equal(t, 11*time.Hour, target.Sub(now))
	})
}
