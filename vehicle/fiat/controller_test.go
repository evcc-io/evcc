package fiat

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// makeTime constructs a time.Time on an arbitrary fixed date in UTC
// using the provided hour and minute. It is used to simulate "now" values
// in unit tests without caring about the actual day.
func makeTime(t ...int) time.Time {
	h, m, s := 0, 0, 0
	if len(t) > 0 {
		h = t[0]
	}
	if len(t) > 1 {
		m = t[1]
	}
	if len(t) > 2 {
		s = t[2]
	}
	return time.Date(2026, 7, 8, h, m, s, 0, time.UTC)
}

func newController() *Controller {
	return &Controller{log: util.NewLogger("fiat-test")}
}

func TestRoundUpTo(t *testing.T) {
	interval := 5 * time.Minute

	cases := []struct {
		timeInput   time.Time
		expected    time.Time
		description string
	}{
		{makeTime(10, 0), makeTime(10, 5), "aligned to boundary"},
		{makeTime(10, 1), makeTime(10, 5), "one minute past"},
		{makeTime(10, 5), makeTime(10, 10), "exact boundary moves forward"},
		{makeTime(10, 4, 59), makeTime(10, 5, 0), "seconds before boundary"},
	}

	for _, c := range cases {
		res := roundUpTo(interval, c.timeInput)
		assert.Equal(t, c.expected, res, c.description)
	}
}

func TestConfigureChargeSchedule_EndBumpDelay(t *testing.T) {
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "19:40",
	}

	// when now is only one minute past the original end, it should not be
	// bumped (half of 5m = 2.5m threshold)
	has := c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 41))
	assert.False(t, has, "unexpected change for too-soon update")
	assert.Equal(t, "19:40", schedule.EndTime)

	// once we cross the half-interval threshold (>= 2.5m after old end)
	// the schedule should be updated to the next 5‑minute boundary
	schedule.EndTime = "19:40" // reset for clarity
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 43))
	assert.True(t, has, "expected schedule to be bumped")
	assert.Equal(t, "19:45", schedule.EndTime)
}

func TestConfigureChargeSchedule_EndEarlierThanCurrent(t *testing.T) {
	// if the requested end time is before the current end, we should still
	// honour it immediately (e.g. user shortened the charge window)
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "20:00",
	}

	// it will round to 19:55 and update.
	now := makeTime(19, 54)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "19:55", schedule.EndTime)

	// it will round to 20:00 and not update.
	schedule.EndTime = "20:00" // reset for clarity
	now = makeTime(19, 58)
	has = c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.False(t, has)
	assert.Equal(t, "20:00", schedule.EndTime)
}

func TestConfigureChargeSchedule_ParseErrorFallback(t *testing.T) {
	// if the existing EndTime cannot be parsed we should still perform the
	// update rather than crash or skip.
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "garbage",
	}

	now := makeTime(19, 43)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "19:45", schedule.EndTime)
}

func TestConfigureChargeSchedule_StartAfterEnd(t *testing.T) {
	// if start time is after end time (schedule crossing midnight), start
	// should be set to the fallback value "00:00" to avoid API rejections
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "22:00",
		EndTime:            "23:55",
	}

	// trigger validation by changing the end time (which will be parsed and
	// compared against the start time)
	now := makeTime(9, 43)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "00:00", schedule.StartTime, "start should be set to fallback when after end")
	assert.Equal(t, "09:45", schedule.EndTime)
}

func TestConfigureChargeSchedule_StartEqualEnd(t *testing.T) {
	// if start time equals end time, it means charge was stopped before the
	// schedule start time; the schedule should be disabled to avoid unwanted
	// charge start.
	// We test this by setting start and end to the same value through updates.
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "20:00",
	}

	// Set both start and end to values that round to the same time (19:05).
	// First update: set start to 19:03 which rounds to 19:05
	now := makeTime(19, 3)
	has := c.configureChargeSchedule(schedule, now, time.Time{})
	assert.True(t, has)
	assert.Equal(t, "19:05", schedule.StartTime)

	// Second update: set end to 19:04 which also rounds to 19:05
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 4))
	assert.True(t, has)
	assert.False(t, schedule.EnableScheduleType, "schedule should be disabled when start equals end")
	assert.Equal(t, "19:05", schedule.StartTime)
	assert.Equal(t, "19:05", schedule.EndTime)
}

func TestConfigureChargeSchedule_StopBeforeStart(t *testing.T) {
	// if stop charge happens before the schedule start time
	// and end time ends up before start time due to the different rounding logic
	// schedule should still be consistant and valid
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "20:00",
	}

	// Set both start and end to values that round to the same time (19:05).
	// First update: set start to 19:03 which rounds to 19:05
	now := makeTime(19, 2, 0)
	has := c.configureChargeSchedule(schedule, now, time.Time{})
	assert.True(t, has)
	assert.Equal(t, "19:05", schedule.StartTime)

	// Second update: set end right after which rounds to 19:00
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 2, 10))
	// Start & end time should be consistent and valid, and in the past to avoid charge starting
	assert.Equal(t, "00:00", schedule.StartTime)
	assert.Equal(t, "19:00", schedule.EndTime)
}

func TestConfigureChargeSchedule_ParseErrorStartOnly(t *testing.T) {
	// if only the start time fails to parse, it should be set to fallback
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "bad_time",
		EndTime:            "19:40",
	}

	now := makeTime(19, 43)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "00:00", schedule.StartTime)
	assert.Equal(t, "19:45", schedule.EndTime)
}

func TestConfigureChargeSchedule_ScheduleTypeEnabling(t *testing.T) {
	// if schedule type is not CHARGE or is disabled, it should be corrected
	// and other settings should be initialized
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "PRECONDITIONING",
		EnableScheduleType: false,
		StartTime:          "10:00",
		EndTime:            "20:00",
	}

	now := makeTime(19, 42)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "CHARGE", schedule.ScheduleType)
	assert.True(t, schedule.EnableScheduleType)
	assert.False(t, schedule.CabinPriority)
	assert.False(t, schedule.ChargeToFull)
	assert.True(t, schedule.RepeatSchedule)
}

func TestConfigureChargeSchedule_ScheduledDaysReset(t *testing.T) {
	// when schedule is changed, scheduled days should be set to only the
	// current day to avoid undesired charge in the future
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "19:00",
	}
	schedule.ScheduledDays.Monday = false
	schedule.ScheduledDays.Tuesday = false
	schedule.ScheduledDays.Wednesday = false
	schedule.ScheduledDays.Thursday = false
	schedule.ScheduledDays.Friday = false
	schedule.ScheduledDays.Saturday = true
	schedule.ScheduledDays.Sunday = true

	// Friday, 2026-02-27
	now := time.Date(2026, 2, 27, 19, 43, 0, 0, time.UTC)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	// Only Friday should be enabled
	assert.False(t, schedule.ScheduledDays.Monday)
	assert.False(t, schedule.ScheduledDays.Tuesday)
	assert.False(t, schedule.ScheduledDays.Wednesday)
	assert.False(t, schedule.ScheduledDays.Thursday)
	assert.True(t, schedule.ScheduledDays.Friday)
	assert.False(t, schedule.ScheduledDays.Saturday)
	assert.False(t, schedule.ScheduledDays.Sunday)
}

func TestConfigureChargeSchedule_NoChangeWhenNoEndOrStart(t *testing.T) {
	// if neither start nor end is provided, schedule should not be modified
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "19:40",
	}

	has := c.configureChargeSchedule(schedule, time.Time{}, time.Time{})
	assert.False(t, has)
	assert.Equal(t, "10:00", schedule.StartTime)
	assert.Equal(t, "19:40", schedule.EndTime)
}
