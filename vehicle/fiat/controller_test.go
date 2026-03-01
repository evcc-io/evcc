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

func TestConfigureChargeSchedule_NominalChargeSession(t *testing.T) {
	// if the requested end time is before the current end, we should still
	// honour it immediately (e.g. user shortened the charge window)
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
	}

	// Start charge
	has := c.configureChargeSchedule(schedule, makeTime(10, 04), time.Time{})
	assert.True(t, has)
	assert.Equal(t, "10:05", schedule.StartTime, "start should be rounded to 5-minute")
	assert.Equal(t, "23:55", schedule.EndTime, "end time should always set to 23:55 when starting charge")
	assert.True(t, schedule.ScheduledDays.Wednesday, "schedule should be enabled for Wednesday")
	assert.False(t, schedule.ScheduledDays.Monday || schedule.ScheduledDays.Tuesday || schedule.ScheduledDays.Thursday ||
		schedule.ScheduledDays.Friday || schedule.ScheduledDays.Saturday || schedule.ScheduledDays.Sunday, "schedule should be false for all other days")

	// Stop charge few hours later on the same day
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(14, 02))
	assert.True(t, has)
	assert.Equal(t, "10:05", schedule.StartTime, "start should not change when stopping charge")
	assert.Equal(t, "14:00", schedule.EndTime, "end time should be rounded to 5-minute when stopping charge")
	assert.True(t, schedule.ScheduledDays.Wednesday, "schedule should be enabled for Wednesday")
	assert.False(t, schedule.ScheduledDays.Monday || schedule.ScheduledDays.Tuesday || schedule.ScheduledDays.Thursday ||
		schedule.ScheduledDays.Friday || schedule.ScheduledDays.Saturday || schedule.ScheduledDays.Sunday, "schedule should be false for all other days")
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

	// Start charge must enable schedule type and set it to CHARGE, and reset other settings to defaults
	has := c.configureChargeSchedule(schedule, makeTime(19, 42), time.Time{})
	assert.True(t, has)
	assert.Equal(t, "CHARGE", schedule.ScheduleType, "schedule type should be set to CHARGE")
	assert.True(t, schedule.EnableScheduleType, "schedule should be enabled")
	assert.False(t, schedule.CabinPriority, "cabin priority should be false")
	assert.False(t, schedule.ChargeToFull, "charge to full should be false")
	assert.True(t, schedule.RepeatSchedule, "repeat schedule should be true")
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

func TestConfigureChargeSchedule_ParseErrorStartOnly(t *testing.T) {
	// if only the start time fails to parse, it should be set to fallback
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "bad_time",
		EndTime:            "23:55",
	}

	// Set end time with a invalid start time
	has := c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 43))
	assert.True(t, has)
	assert.Equal(t, "00:00", schedule.StartTime)
	assert.Equal(t, "19:45", schedule.EndTime)
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
	has := c.configureChargeSchedule(schedule, time.Date(2026, 2, 27, 19, 43, 0, 0, time.UTC), time.Time{})
	assert.True(t, has)
	// Only Friday should be enabled after start
	assert.False(t, schedule.ScheduledDays.Monday, "Monday should be disabled")
	assert.False(t, schedule.ScheduledDays.Tuesday, "Tuesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Wednesday, "Wednesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Thursday, "Thursday should be disabled")
	assert.True(t, schedule.ScheduledDays.Friday, "Friday should be enabled")
	assert.False(t, schedule.ScheduledDays.Saturday, "Saturday should be disabled")
	assert.False(t, schedule.ScheduledDays.Sunday, "Sunday should be disabled")

	// Set end time, which should not change the scheduled days as they were already set to current day on start time update
	has = c.configureChargeSchedule(schedule, time.Time{}, time.Date(2026, 2, 27, 19, 50, 0, 0, time.UTC))
	assert.True(t, has)
	assert.False(t, schedule.ScheduledDays.Monday, "Monday should be disabled")
	assert.False(t, schedule.ScheduledDays.Tuesday, "Tuesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Wednesday, "Wednesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Thursday, "Thursday should be disabled")
	assert.True(t, schedule.ScheduledDays.Friday, "Friday should be enabled")
	assert.False(t, schedule.ScheduledDays.Saturday, "Saturday should be disabled")
	assert.False(t, schedule.ScheduledDays.Sunday, "Sunday should be disabled")

	// Next day is Saturday, 2026-02-28: if we start the schedule again on the next day, only Saturday should be enabled
	has = c.configureChargeSchedule(schedule, time.Date(2026, 2, 28, 8, 12, 0, 0, time.UTC), time.Time{})
	assert.True(t, has, "expected schedule to be updated for new day")
	// Only Saturday should be enabled
	assert.False(t, schedule.ScheduledDays.Monday, "Monday should be disabled")
	assert.False(t, schedule.ScheduledDays.Tuesday, "Tuesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Wednesday, "Wednesday should be disabled")
	assert.False(t, schedule.ScheduledDays.Thursday, "Thursday should be disabled")
	assert.False(t, schedule.ScheduledDays.Friday, "Friday should be disabled")
	assert.True(t, schedule.ScheduledDays.Saturday, "Saturday should be enabled")
	assert.False(t, schedule.ScheduledDays.Sunday, "Sunday should be disabled")
}

func TestConfigureChargeSchedule_CrossingMidnight(t *testing.T) {
	// if start time is after end time (schedule crossing midnight), start
	// should be set to the fallback value "00:00" to avoid API rejections
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "22:00",
		EndTime:            "23:55",
	}
	// Fix weekday for test is a Wednesday, so for this test the previous scheduled day is Tuesday
	schedule.ScheduledDays.Monday = false
	schedule.ScheduledDays.Tuesday = true
	schedule.ScheduledDays.Wednesday = false
	schedule.ScheduledDays.Thursday = false
	schedule.ScheduledDays.Friday = false
	schedule.ScheduledDays.Saturday = false
	schedule.ScheduledDays.Sunday = false

	// trigger validation by changing the end time (which will be parsed and compared against the start time)
	now := makeTime(7, 15)
	has := c.configureChargeSchedule(schedule, time.Time{}, now)
	assert.True(t, has)
	assert.Equal(t, "00:00", schedule.StartTime, "start should be set to fallback when after end")
	assert.Equal(t, "07:15", schedule.EndTime)
	assert.False(t, schedule.ScheduledDays.Tuesday, "Tuesday should be disabled")
	assert.True(t, schedule.ScheduledDays.Wednesday, "Wednesday should be enabled")
}

func TestConfigureChargeSchedule_AvoidEndlessEndPostpone(t *testing.T) {
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
		StartTime:          "10:00",
		EndTime:            "19:40",
	}

	// when now is only one minute past the original end, end should not be postponed
	has := c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 41, 45))
	assert.False(t, has, "unexpected change for too-soon update")
	assert.Equal(t, "19:40", schedule.EndTime)

	// once we cross the rouding threshold the schedule should be updated to the next 5‑minute boundary
	schedule.EndTime = "19:40" // reset for clarity
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 43))
	assert.True(t, has, "expected schedule to be bumped")
	assert.Equal(t, "19:45", schedule.EndTime)
}

func TestConfigureChargeSchedule_StartStopStartAgainInShortTime(t *testing.T) {
	// if start time equals end time, it means charge was stopped right before or right after the schedule start time.
	// If after this, we want to start charge again, we need to make sure the schedule is correctly re-enabled
	// by setting end time to default value when enabling charge.
	c := newController()

	schedule := &Schedule{
		ScheduleType:       "CHARGE",
		EnableScheduleType: true,
	}

	// Set both start and end to values that round to the same time (19:05).
	// First update: set start to 19:03 which rounds to 19:05
	has := c.configureChargeSchedule(schedule, makeTime(19, 3), time.Time{})
	assert.True(t, has)
	assert.Equal(t, "19:05", schedule.StartTime)
	assert.Equal(t, "23:55", schedule.EndTime) // Default end time should always be set when enabling charge

	// Second update: set end to 19:04 which also rounds to 19:05
	has = c.configureChargeSchedule(schedule, time.Time{}, makeTime(19, 4))
	assert.True(t, has)
	assert.Equal(t, "19:05", schedule.StartTime)
	assert.Equal(t, "19:05", schedule.EndTime)

	// Start charge again: few seconds before schedule start time
	has = c.configureChargeSchedule(schedule, makeTime(19, 4, 30), time.Time{})
	assert.True(t, has)
	assert.Equal(t, "19:05", schedule.StartTime)
	assert.Equal(t, "23:55", schedule.EndTime) // Default end time should always be set when enabling charge

	// Start charge again: few seconds after schedule start time
	has = c.configureChargeSchedule(schedule, makeTime(19, 5, 15), time.Time{})
	assert.False(t, has, "unexpected change when start again right after schedule start")
	assert.Equal(t, "19:05", schedule.StartTime)
	assert.Equal(t, "23:55", schedule.EndTime) // Default end time should always be set when enabling charge
}
