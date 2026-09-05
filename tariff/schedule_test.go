package tariff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleTimerDuration(t *testing.T) {
	// empty -> fallback applies
	timer, err := schedule{}.timer(3 * time.Hour)
	require.NoError(t, err)
	assert.Nil(t, timer.sched)
	assert.Equal(t, 3*time.Hour, timer.interval)
	assert.Equal(t, 6*time.Hour, timer.window())

	// duration form wins over fallback
	timer, err = schedule{Interval: "30m"}.timer(time.Hour)
	require.NoError(t, err)
	assert.Nil(t, timer.sched)
	assert.Equal(t, 30*time.Minute, timer.interval)
}

func TestScheduleTimerDurationInvalid(t *testing.T) {
	_, err := schedule{Interval: "0s"}.timer(time.Hour)
	require.Error(t, err)
}

func TestScheduleTimerCron(t *testing.T) {
	timer, err := schedule{Interval: "15 0 * * *"}.timer(time.Hour)
	require.NoError(t, err)
	require.NotNil(t, timer.sched)

	// next fire is at 00:15 local time
	base := time.Date(2026, 8, 23, 12, 0, 0, 0, time.Local)
	next := timer.sched.Next(base)
	assert.Equal(t, 0, next.Hour())
	assert.Equal(t, 15, next.Minute())
	assert.Equal(t, base.AddDate(0, 0, 1).Day(), next.Day())
}

func TestScheduleTimerCronDescriptor(t *testing.T) {
	timer, err := schedule{Interval: "@daily"}.timer(time.Hour)
	require.NoError(t, err)
	require.NotNil(t, timer.sched)
}

func TestScheduleTimerCronImpossible(t *testing.T) {
	// syntactically valid but never matches (Feb 30) -> zero next-fire.
	// Must be rejected so C() cannot busy-loop.
	_, err := schedule{Interval: "0 0 30 2 *"}.timer(time.Hour)
	require.Error(t, err)
}

func TestScheduleTimerInvalid(t *testing.T) {
	_, err := schedule{Interval: "nonsense"}.timer(time.Hour)
	require.Error(t, err)
}

func TestScheduleTimerStale(t *testing.T) {
	// duration-based: stale once interval has elapsed since updated
	timer, err := schedule{Interval: "1h"}.timer(time.Hour)
	require.NoError(t, err)
	assert.False(t, timer.stale(time.Now().Add(-30*time.Minute)))
	assert.True(t, timer.stale(time.Now().Add(-2*time.Hour)))

	// cron-based: stale once a scheduled fire has passed since updated
	timer, err = schedule{Interval: "15 0 * * *"}.timer(time.Hour)
	require.NoError(t, err)
	// updated a minute ago -> next 00:15 not yet reached (unless run at 00:14)
	assert.False(t, timer.stale(time.Now().Add(-time.Minute)))
	// updated more than a day ago -> a 00:15 fire has certainly passed
	assert.True(t, timer.stale(time.Now().Add(-25*time.Hour)))
}

func TestScheduleTimerC(t *testing.T) {
	timer, err := schedule{Interval: "10ms"}.timer(time.Hour)
	require.NoError(t, err)

	select {
	case <-timer.C():
	case <-time.After(time.Second):
		t.Fatal("timer did not fire")
	}
}
