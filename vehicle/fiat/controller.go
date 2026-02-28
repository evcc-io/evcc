package fiat

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Controller struct {
	pvd *Provider
	api *API
	log *util.Logger
	vin string
	pin string
}

// NewController creates a vehicle current and charge controller
func NewController(provider *Provider, api *API, log *util.Logger, vin string, pin string) *Controller {
	impl := &Controller{
		pvd: provider,
		api: api,
		log: log,
		vin: vin,
		pin: pin,
	}
	return impl
}

var _ api.CurrentController = (*Controller)(nil)

// MaxCurrent implements the api.CurrentController interface
func (c *Controller) MaxCurrent(current int64) error {
	// Even if we cannot control the current, this interface must be implemented otherwise the ChargeEnable is never called
	return nil
}

var _ api.ChargeController = (*Controller)(nil)

// ChargeEnable implements the api.ChargeController interface
func (c *Controller) ChargeEnable(enable bool) error {
	if c.pin == "" {
		return api.ErrMissingCredentials
	}

	// get current schedule status from provider (cached)
	stat, err := c.pvd.statusG()
	if err != nil {
		return err
	}
	if stat.EvInfo == nil || stat.EvInfo.Schedules == nil || len(stat.EvInfo.Schedules) == 0 {
		return api.ErrNotAvailable
	}

	hasChanged := false // Will track if we made any change to the schedule to avoid unnecessary updates through API call
	now := time.Now()   // Call once and reuse

	if enable {
		// Start charging from now until end of day (23:55)
		hasChanged = hasChanged || c.configureChargeSchedule(&stat.EvInfo.Schedules[0], now, time.Time{})
	} else {
		// Stop charging: set charge end time to now to stop charging as soon as possible; use empty time to keep start time as it was for history in Fiat app
		hasChanged = hasChanged || c.configureChargeSchedule(&stat.EvInfo.Schedules[0], time.Time{}, now)
	}

	// make sure the other charge schedules are disabled in case user changed them
	hasChanged = hasChanged || c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[1])
	hasChanged = hasChanged || c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[2])

	// post new schedule, but only if something changed to avoid unnecessary API calls
	if hasChanged {
		res, err := c.api.UpdateSchedule(c.vin, c.pin, stat.EvInfo.Schedules)
		if err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}
		if res.ResponseStatus != "pending" {
			return fmt.Errorf("invalid response status: %s", res.ResponseStatus)
		}
		c.log.DEBUG.Printf("updated first charge schedule: enable=%v, start=%s, end=%s",
			enable, stat.EvInfo.Schedules[0].StartTime, stat.EvInfo.Schedules[0].EndTime)
	}

	return nil
}

// Helper to set the correct schedule days matching the provided time, to ensure the schedule is only applied for the current day when it's updated
func setScheduleDays(schedule *Schedule, t time.Time) {
	weekday := t.Weekday()
	schedule.ScheduledDays.Monday = (weekday == time.Monday)
	schedule.ScheduledDays.Tuesday = (weekday == time.Tuesday)
	schedule.ScheduledDays.Wednesday = (weekday == time.Wednesday)
	schedule.ScheduledDays.Thursday = (weekday == time.Thursday)
	schedule.ScheduledDays.Friday = (weekday == time.Friday)
	schedule.ScheduledDays.Saturday = (weekday == time.Saturday)
	schedule.ScheduledDays.Sunday = (weekday == time.Sunday)
}

// configureChargeSchedule configures the provided schedule with the provided start and end time, while ensuring it fits API requirements and avoiding unnecessary.
// It returns true if the schedule was changed and false otherwise.
func (c *Controller) configureChargeSchedule(schedule *Schedule, start time.Time, end time.Time) bool {
	const (
		minTimeInterval   = 5 * time.Minute // Minimum time interval accepted by Fiat API in schedules; used for rounding start and end time to avoid API rejections
		timeFormat        = "15:04"         // Hours & minutes only
		defaultEndTime    = "23:55"         // Default end time to use when enabling charge; this is the last time of the day accepted by the Fiat API
		fallbackStartTime = "00:00"         // Fallback time for schedules crossing midnight; this is the first time of the day accepted by the Fiat API
	)

	hasChanged := false // track if we made any change to the schedule to avoid unnecessary API calls

	// Make sure schedule is enabled and of type CHARGE
	if schedule.ScheduleType != "CHARGE" || !schedule.EnableScheduleType {
		schedule.ScheduleType = "CHARGE"
		schedule.EnableScheduleType = true
		schedule.CabinPriority = false
		schedule.ChargeToFull = false
		schedule.RepeatSchedule = true
		hasChanged = true
		c.log.DEBUG.Printf("schedule type changed to CHARGE and enabled")
	}

	// Update start only if provided (non-zero)
	if !start.IsZero() {
		// round to 5 minutes to avoid API rejections, and allow trying to start in few minutes in the past to start as soon as possible
		newStartStr := start.Round(minTimeInterval).Format(timeFormat)

		// Update only if different from current
		if newStartStr != schedule.StartTime {
			schedule.StartTime = newStartStr
			schedule.EndTime = defaultEndTime // Set default end time when enabling charge to avoid API rejections for schedules without end time
			setScheduleDays(schedule, start)  // Set schedule days matching the provided start time to ensure the schedule is only applied for the current day when it's updated
			hasChanged = true
			c.log.DEBUG.Printf("set charge schedule start: %s with default end time: %s", schedule.StartTime, schedule.EndTime)
		} else if schedule.StartTime == schedule.EndTime {
			// If start and end are the same, it means it was previously disabled
			// We want to start charge again => we need to set end time to default value to make sure the schedule is enabled
			schedule.EndTime = defaultEndTime // Set default end time when enabling charge to avoid API rejections for schedules without end time
			setScheduleDays(schedule, start)  // Set schedule days matching the provided start time to ensure the schedule is only applied for the current day when it's updated
			hasChanged = true
			c.log.DEBUG.Printf("schedule re-enabled because start and end times were equal")
		}
	}

	// Update end only if provided (non-zero)
	if !end.IsZero() {
		// round to 5 minutes to avoid API rejections, and allow some delay to stop charge (the round down is the delay)
		newEndStr := end.Round(minTimeInterval).Format(timeFormat)

		// Update only if different from current
		if newEndStr != schedule.EndTime {
			schedule.EndTime = newEndStr
			hasChanged = true
			c.log.DEBUG.Printf("set charge schedule end: %s", schedule.EndTime)
		}
	}

	// If one of the time changed, make sure the schedule is always consistent even in edge cases.
	if (!start.IsZero() || !end.IsZero()) && hasChanged {
		// To ensure proper comparison of times, we need to parse them back from string to time.Time.
		chkStart, err1 := time.Parse(timeFormat, schedule.StartTime)
		chkEnd, err2 := time.Parse(timeFormat, schedule.EndTime)
		if err1 != nil || err2 != nil {
			c.log.WARN.Printf("failed to parse schedule times: start=%v, end=%v", err1, err2)
			if err1 != nil {
				// If start time cannot be parsed, set to fallback value
				schedule.StartTime = fallbackStartTime
				setScheduleDays(schedule, end) // Set schedule days matching the provided end time to ensure the schedule is only applied for the current day when it's updated
				hasChanged = true
				c.log.DEBUG.Printf("set charge schedule start to fallback value %s due to parse error", fallbackStartTime)
			}
			if err2 != nil {
				// If end time cannot be parsed, set to default end time
				schedule.EndTime = defaultEndTime
				hasChanged = true
				c.log.DEBUG.Printf("set charge schedule end to default value %s due to parse error", defaultEndTime)
			}
		} else if chkStart.After(chkEnd) {
			// If start time is after end time (can only happen when setting end), set start time to fallback value to avoid API rejections for schedules crossing midnight
			c.log.DEBUG.Printf("start time %s is after end time %s, setting start time to fallback value %s", schedule.StartTime, schedule.EndTime, fallbackStartTime)
			schedule.StartTime = fallbackStartTime
			setScheduleDays(schedule, end) // Set schedule days matching the provided end time to ensure the schedule is only applied for the current day when it's updated
			hasChanged = true
		} else if chkStart.Equal(chkEnd) {
			// If start time is equal to end time, it means the charge has been stopped before the schedule start time => disable it to avoid charge to start
			c.log.DEBUG.Printf("start time %s is equal to end time %s, disabling schedule", schedule.StartTime, schedule.EndTime)
			schedule.EnableScheduleType = false
			hasChanged = true
		}
	}

	return hasChanged
}

// disableConflictingChargeSchedule makes sure the provided schedule is disabled if it's of type CHARGE to avoid conflicts between schedules and potential API rejections for conflicting schedules. It returns true if the schedule was changed and false otherwise.
func (c *Controller) disableConflictingChargeSchedule(schedule *Schedule) bool {
	// make sure the other charge schedules are disabled in case user changed them
	if schedule.ScheduleType == "CHARGE" && schedule.EnableScheduleType {
		schedule.EnableScheduleType = false
		c.log.DEBUG.Printf("disabled charge schedule other than the first one to avoid conflicts")
		return true // schedule was changed
	}
	return false // schedule was not changed
}

var _ api.Resurrector = (*Controller)(nil)

func (c *Controller) WakeUp() error {
	if c.pin == "" {
		c.log.DEBUG.Printf("vehicle cannot be woken up: no PIN provided")
		return nil
	}

	// get current schedule status from provider (cached)
	stat, err := c.pvd.statusG()
	if err == nil && stat.EvInfo != nil && stat.EvInfo.Schedules != nil && len(stat.EvInfo.Schedules) > 0 &&
		stat.EvInfo.Schedules[0].EnableScheduleType && stat.EvInfo.Schedules[0].ScheduleType == "CHARGE" {
		// If the first schedule is already enabled for charge, don't go further to avoid chargeNow forcing immediate charge start and messing up with schedules
		c.log.DEBUG.Printf("vehicle wakeup skipped because charge schedule is already enabled, to avoid conflicts with schedules")
		return nil
	}

	// No charge schedule is set and we need to wakeup the vehicle as charge is not starting => let's call ChargeNow to start the charge
	res, err := c.api.ChargeNow(c.vin, c.pin)
	if err != nil {
		return fmt.Errorf("charge now call failed: %w", err)
	}
	if res.ResponseStatus != "pending" {
		return fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}
	c.log.DEBUG.Printf("vehicle wakeup triggered successfully with charge now action")

	return nil
}
