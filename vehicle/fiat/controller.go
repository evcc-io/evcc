package fiat

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
)

type Controller struct {
	pvd              *Provider
	api              *API
	vin              string
	pin              string
	requestedCurrent int64
}

// NewController creates a vehicle current and charge controller
func NewController(provider *Provider, api *API, vin string, pin string) *Controller {
	impl := &Controller{
		pvd:              provider,
		api:              api,
		vin:              vin,
		pin:              pin,
		requestedCurrent: 0,
	}
	return impl
}

var _ api.CurrentController = (*Controller)(nil)

// MaxCurrent implements the api.CurrentController interface
func (c *Controller) MaxCurrent(current int64) error {
	// Even if we cannot control the current, this interface must be implemented otherwise the ChargeEnable is never called
	// Store the requested current
	c.requestedCurrent = current
	return nil
}

var _ api.CurrentGetter = (*Controller)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *Controller) GetMaxCurrent() (float64, error) {
	// To avoid errors on evcc as we cannot control the current on the vehicle for now, return always the requested current
	return float64(c.requestedCurrent), nil
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
	c.ConfigureChargeSchedule(&stat.EvInfo.Schedules[0])

	if enable {
		// start charging by updating active charge schedule to start now and end in 12h
		stat.EvInfo.Schedules[0].StartTime = time.Now().Format("15:04")                   // only hour and minutes
		stat.EvInfo.Schedules[0].EndTime = time.Now().Add(time.Hour * 12).Format("15:04") // only hour and minutes
	} else {
		// stop charging by updating active charge schedule end time to now
		stat.EvInfo.Schedules[0].EndTime = time.Now().Format("15:04") // only hour and minutes
	}

	// make sure the other charge schedules are disabled in case user changed them
	c.DisableConflictingChargeSchedule(&stat.EvInfo.Schedules[1])
	c.DisableConflictingChargeSchedule(&stat.EvInfo.Schedules[2])

	// post new schedule
	res, err := c.api.UpdateSchedule(c.vin, c.pin, stat.EvInfo.Schedules)

	if err == nil && res.ResponseStatus != "pending" {
		err = fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}
	return err
}

func (c *Controller) ConfigureChargeSchedule(schedule *ScheduleData) {
	// all values are set to be sure no manual change can lead to inconsistencies
	schedule.CabinPriority = false
	schedule.ChargeToFull = false
	schedule.EnableScheduleType = true
	schedule.RepeatSchedule = true
	schedule.ScheduleType = "CHARGE"
	schedule.ScheduledDays.Friday = true
	schedule.ScheduledDays.Monday = true
	schedule.ScheduledDays.Saturday = true
	schedule.ScheduledDays.Sunday = true
	schedule.ScheduledDays.Thursday = true
	schedule.ScheduledDays.Tuesday = true
	schedule.ScheduledDays.Wednesday = true
}

func (c *Controller) DisableConflictingChargeSchedule(schedule *ScheduleData) {
	// make sure the other charge schedules are disabled in case user changed them
	if schedule.ScheduleType == "CHARGE" && schedule.EnableScheduleType {
		schedule.EnableScheduleType = false
	}
}
