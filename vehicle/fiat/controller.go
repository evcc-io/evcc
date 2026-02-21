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
		// Start charging: configure charge from now (end time will be handled in computeNewApiScheduleTime)
		hasChanged = c.configureChargeSchedule(&stat.EvInfo.Schedules[0], now, time.Time{}) // only set start time to now and keep end time unchanged to avoid undesired charge stop in the future
	} else {
		// Stop charging: update end time (use empty time to keep start time as it was for history in Fiat app)
		hasChanged = c.configureChargeSchedule(&stat.EvInfo.Schedules[0], time.Time{}, now)
	}

	// make sure the other charge schedules are disabled in case user changed them
	hasChanged = hasChanged || c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[1])
	hasChanged = hasChanged || c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[2])

	// post new schedule, but only if something changed to avoid unnecessary API calls
	if hasChanged {
		res, err := c.api.UpdateSchedule(c.vin, c.pin, stat.EvInfo.Schedules)
		c.log.INFO.Printf("updated first charge schedule: enable=%v, start=%s, end=%s", enable, stat.EvInfo.Schedules[0].StartTime, stat.EvInfo.Schedules[0].EndTime)
		if err == nil && res.ResponseStatus != "pending" {
			err = fmt.Errorf("invalid response status: %s", res.ResponseStatus)
		}
	}

	return err
}

// computeNewApiScheduleTime computes the new schedule time to set in the API based on the current schedule time and the target time provided by the user, while ensuring it fits API requirements (rounding up to next 5 minutes boundary and avoiding changes if time difference is not significant to prevent API rejections for unchanged schedules)
func (c *Controller) computeNewApiScheduleTime(current string, target time.Time, timeFormat string) string {

	const (
		minTimeInterval = 5 * time.Minute // Minimum time interval accepted by Fiat API in schedules; used for rounding up start and end time to avoid API rejections
	)

	// By default, return current time unchanged to avoid changing if target time is not significantly different from current time
	result := current

	// Parse previous schedule time to detect if this is a meaningful change
	currentTime, err1 := time.Parse(timeFormat, current)
	targetTime, err2 := time.Parse(timeFormat, target.Format(timeFormat)) // Format target time to same format as current time for proper comparison
	timeDiff := targetTime.Sub(currentTime).Abs()
	c.log.DEBUG.Printf("current schedule time: %s, target time: %s, time difference: %s", currentTime.Format(timeFormat), targetTime.Format(timeFormat), timeDiff)

	// Round up only if end time changed significantly or if parsing previous time failed
	if err1 != nil || err2 != nil || timeDiff > (minTimeInterval/2) {
		// round up to next 5 minutes boundary to avoid API rejections
		roundedTarget := target.Truncate(minTimeInterval)
		if roundedTarget.Before(target) {
			roundedTarget = roundedTarget.Add(minTimeInterval)
		}
		result = roundedTarget.Format(timeFormat)
		c.log.DEBUG.Printf("target time %s rounded to %s to fit API requirements", target.Format(timeFormat), result)
	}

	return result
}

// configureChargeSchedule configures the provided schedule with the provided start and end time, while ensuring it fits API requirements and avoiding unnecessary changes if times are not significantly different to prevent API rejections for unchanged schedules. It returns true if the schedule was changed and false otherwise.
func (c *Controller) configureChargeSchedule(schedule *Schedule, start time.Time, end time.Time) bool {
	const (
		timeFormat        = "15:04" // Hours & minutes only
		defaultEndTime    = "23:55" // Default end time to use when enabling charge to avoid API rejections for schedules without end time; set to end of the day to avoid undesired charge stop in the future
		fallbackStartTime = "00:00" // Fallback time for schedules crossing midnight
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
		newStartStr := c.computeNewApiScheduleTime(schedule.StartTime, start, timeFormat)

		// Update only if different from current
		if newStartStr != schedule.StartTime {
			schedule.StartTime = newStartStr
			schedule.EndTime = defaultEndTime // Set default end time when enabling charge to avoid API rejections for schedules without end time
			hasChanged = true
			c.log.DEBUG.Printf("set charge schedule start: %s with default end time: %s", schedule.StartTime, schedule.EndTime)

			// only enable for current day to avoid undesired charge start in the future
			weekday := time.Now().Weekday()
			schedule.ScheduledDays.Monday = (weekday == time.Monday)
			schedule.ScheduledDays.Tuesday = (weekday == time.Tuesday)
			schedule.ScheduledDays.Wednesday = (weekday == time.Wednesday)
			schedule.ScheduledDays.Thursday = (weekday == time.Thursday)
			schedule.ScheduledDays.Friday = (weekday == time.Friday)
			schedule.ScheduledDays.Saturday = (weekday == time.Saturday)
			schedule.ScheduledDays.Sunday = (weekday == time.Sunday)
		}
	}

	// Update end only if provided (non-zero)
	if !end.IsZero() {
		newEndStr := c.computeNewApiScheduleTime(schedule.EndTime, end, timeFormat)

		// Update only if different from current
		if newEndStr != schedule.EndTime {
			schedule.EndTime = newEndStr
			hasChanged = true
			c.log.DEBUG.Printf("set charge schedule end: %s", schedule.EndTime)
		}
	}

	// If one of the time changed, make sure start time is always before end time (parse both from string to ensure proper comparison)
	if (!start.IsZero() || !end.IsZero()) && hasChanged {
		chkStart, err1 := time.Parse(timeFormat, schedule.StartTime)
		chkEnd, err2 := time.Parse(timeFormat, schedule.EndTime)
		if err1 == nil && err2 == nil && chkStart.After(chkEnd) {
			// If start time is after end time, set start time to fallback value (00:01) to avoid API rejections for schedules crossing midnight
			c.log.DEBUG.Printf("start time %s is after end time %s, setting start time to fallback value %s", schedule.StartTime, schedule.EndTime, fallbackStartTime)
			schedule.StartTime = fallbackStartTime
			hasChanged = true
		} else if err1 != nil || err2 != nil {
			c.log.WARN.Printf("failed to parse schedule times: start=%v, end=%v", err1, err2)
			if err1 != nil {
				// If start time cannot be parsed, also set to fallback value
				schedule.StartTime = fallbackStartTime
				hasChanged = true
				c.log.DEBUG.Printf("set charge schedule start to fallback value %s due to parse error", fallbackStartTime)
			}
		}
	}

	return hasChanged
}

// disableConflictingChargeSchedule makes sure the provided schedule is disabled if it's of type CHARGE to avoid conflicts between schedules and potential API rejections for conflicting schedules. It returns true if the schedule was changed and false otherwise.
func (c *Controller) disableConflictingChargeSchedule(schedule *Schedule) bool {
	// make sure the other charge schedules are disabled in case user changed them
	if schedule.ScheduleType == "CHARGE" && schedule.EnableScheduleType {
		schedule.EnableScheduleType = false
		c.log.INFO.Printf("disabled charge schedule other than the first one to avoid conflicts")
		return true // schedule was changed
	}
	return false // schedule was not changed
}

var _ api.Resurrector = (*Controller)(nil)

func (c *Controller) WakeUp() error {
	if c.pin == "" {
		c.log.DEBUG.Printf("pin required for vehicle wakeup")
		return nil
	}

	res, err := c.api.ChargeNow(c.vin, c.pin)
	if err == nil && res.ResponseStatus != "pending" {
		err = fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}

	return err
}
