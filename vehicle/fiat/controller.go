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

	if enable {
		// start charging by updating active charge schedule to start now and end in 12h
		stat.EvInfo.Schedules[0].StartTime = time.Now().Format("15:04")                   // only hour and minutes
		stat.EvInfo.Schedules[0].EndTime = time.Now().Add(time.Hour * 12).Format("15:04") // only hour and minutes
	} else {
		// stop charging by updating active charge schedule end time to now
		stat.EvInfo.Schedules[0].EndTime = time.Now().Format("15:04") // only hour and minutes
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
