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
	c.configureChargeSchedule(&stat.EvInfo.Schedules[0])

	const (
		timeFormat        = "15:04" // Hours & minutes only
		fallbackStartTime = "00:01" // Fallback time for schedules crossing midnight
	)

	currentTime := time.Now() // Call once and reuse

	if enable {
		// Start charging: update active schedule with current time and end time (12h later)
		stat.EvInfo.Schedules[0].StartTime = currentTime.Format(timeFormat)
		stat.EvInfo.Schedules[0].EndTime = currentTime.Add(12 * time.Hour).Format(timeFormat)
	} else {
		// Stop charging: update end time to current time
		stat.EvInfo.Schedules[0].EndTime = currentTime.Format(timeFormat)

		// Parse times for comparison and handle edge case: StartTime > EndTime
		start, err1 := time.Parse(timeFormat, stat.EvInfo.Schedules[0].StartTime)

	if err1 == nil && start.After(currentTime) {
		stat.EvInfo.Schedules[0].StartTime = fallbackStartTime
		}
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

func (c *Controller) configureChargeSchedule(schedule *Schedule) {
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
