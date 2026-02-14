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

	// configure first schedule and make sure it's active
	now := time.Now() // Call once and reuse

	if enable {
		// Start charging: configure charge from now to 12h later
		in12hours := now.Add(12 * time.Hour)
		c.configureChargeSchedule(&stat.EvInfo.Schedules[0], now, in12hours)
	} else {
		// Stop charging: update end time (use empty time to keep start time as it was for history in Fiat app)
		c.configureChargeSchedule(&stat.EvInfo.Schedules[0], time.Time{}, now)
	}

	// make sure the other charge schedules are disabled in case user changed them
	c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[1])
	c.disableConflictingChargeSchedule(&stat.EvInfo.Schedules[2])

	// post new schedule
	res, err := c.api.UpdateSchedule(c.vin, c.pin, stat.EvInfo.Schedules)
	if err == nil && res.ResponseStatus != "pending" {
		err = fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}

	return err
}

func roundUpTo(d time.Duration, t time.Time) time.Time {
	// Round up to next d boundary
	rt := t.Truncate(d)
	if !rt.After(t) {
		rt = rt.Add(d)
	}
	return rt
}

func (c *Controller) configureChargeSchedule(schedule *Schedule, start time.Time, end time.Time) {
	const (
		timeFormat        = "15:04"         // Hours & minutes only
		fallbackStartTime = "00:01"         // Fallback time for schedules crossing midnight
		minTimeInterval   = 5 * time.Minute // Minimum time interval accepted by Fiat API in schedules; used for rounding up start and end time to avoid API rejections
	)

	// all values are set to be sure no manual change can lead to inconsistencies
	schedule.CabinPriority = false
	schedule.ChargeToFull = false
	schedule.EnableScheduleType = true
	schedule.RepeatSchedule = true
	schedule.ScheduleType = "CHARGE"

	// only enable for current day to avoid undesired charge start in the future
	weekday := time.Now().Weekday()
	schedule.ScheduledDays.Monday = (weekday == time.Monday)
	schedule.ScheduledDays.Tuesday = (weekday == time.Tuesday)
	schedule.ScheduledDays.Wednesday = (weekday == time.Wednesday)
	schedule.ScheduledDays.Thursday = (weekday == time.Thursday)
	schedule.ScheduledDays.Friday = (weekday == time.Friday)
	schedule.ScheduledDays.Saturday = (weekday == time.Saturday)
	schedule.ScheduledDays.Sunday = (weekday == time.Sunday)

	// Update start only if provided (non-zero)
	if !start.IsZero() {
		// TODO: test with out rounding & round DOWN. Round up only as last resort.
		//rounded := roundUpTo(minTimeInterval, *start)
		schedule.StartTime = start.Format(timeFormat)
		c.log.DEBUG.Printf("set charge schedule start: %s", schedule.StartTime)
	}

	// Update end only if provided (non-zero); round up to next 5 minutes boundary to increase chance of API accepting the schedule the first time
	if !end.IsZero() {
		rounded := roundUpTo(minTimeInterval, end)
		schedule.EndTime = rounded.Format(timeFormat)
		c.log.DEBUG.Printf("set charge schedule end: %s (rounded from %s)", schedule.EndTime, end.Format(timeFormat))
	}

	// If one of the time changed, make sure start time is always before end time (parse both from string to ensure proper comparison)
	if !start.IsZero() || !end.IsZero() {
		chkStart, err1 := time.Parse(timeFormat, schedule.StartTime)
		chkEnd, err2 := time.Parse(timeFormat, schedule.EndTime)
		if err1 == nil && err2 == nil && chkStart.After(chkEnd) {
			// If start time is after end time, set start time to fallback value (00:01) to avoid API rejections for schedules crossing midnight
			c.log.DEBUG.Printf("start time %s is after end time %s, setting start time to fallback value %s", schedule.StartTime, schedule.EndTime, fallbackStartTime)
			schedule.StartTime = fallbackStartTime
		} else if err1 != nil || err2 != nil {
			c.log.WARN.Printf("failed to parse schedule times: start=%v, end=%v", err1, err2)
			if err1 != nil {
				// If start time cannot be parsed, also set to fallback value
				schedule.StartTime = fallbackStartTime
			}
		}
	}
}

func (c *Controller) disableConflictingChargeSchedule(schedule *Schedule) {
	// make sure the other charge schedules are disabled in case user changed them
	if schedule.ScheduleType == "CHARGE" && schedule.EnableScheduleType {
		schedule.EnableScheduleType = false
	}
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
