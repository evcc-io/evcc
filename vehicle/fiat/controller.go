package fiat

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
)

type Controller struct {
	api              *API
	vin              string
	pin              string
	requestedCurrent int64
	lastStartCharge  time.Time
}

// NewController creates a vehicle current and charge controller
func NewController(api *API, vin string, pin string) *Controller {
	impl := &Controller{
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

	if enable {
		// update charge schedule to start now
		c.lastStartCharge = time.Now()
		return c.ChangeScheduleCharge(c.lastStartCharge, c.lastStartCharge.Add(time.Hour*12))
	} else {
		// Simulate stop charging by updating charge schedule end time to now
		return c.ChangeScheduleCharge(c.lastStartCharge, time.Now())
	}
}

func (c *Controller) ChangeScheduleCharge(startTime time.Time, endTime time.Time) error {
	// get current schedule
	stat, err := c.api.Status(c.vin)
	if err != nil {
		return err
	}
	if stat.EvInfo == nil || stat.EvInfo.Schedules == nil || len(stat.EvInfo.Schedules) == 0 {
		return api.ErrNotAvailable
	}

	// update schedule 1 and make sure it's active
	// all values are set to be sure no manual change can lead to inconsistencies
	stat.EvInfo.Schedules[0].CabinPriority = false
	stat.EvInfo.Schedules[0].ChargeToFull = false
	stat.EvInfo.Schedules[0].EnableScheduleType = true
	stat.EvInfo.Schedules[0].EndTime = endTime.Format("15:04") // only hour and minutes
	stat.EvInfo.Schedules[0].RepeatSchedule = true
	stat.EvInfo.Schedules[0].ScheduleType = "CHARGE"
	stat.EvInfo.Schedules[0].ScheduledDays.Friday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Monday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Saturday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Sunday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Thursday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Tuesday = true
	stat.EvInfo.Schedules[0].ScheduledDays.Wednesday = true
	stat.EvInfo.Schedules[0].StartTime = startTime.Format("15:04") // only hour and minutes

	// make sure the other charge schedules are disabled in case user changed them
	if stat.EvInfo.Schedules[1].ScheduleType == "CHARGE" {
		stat.EvInfo.Schedules[1].EnableScheduleType = false
	}
	if stat.EvInfo.Schedules[2].ScheduleType == "CHARGE" {
		stat.EvInfo.Schedules[2].EnableScheduleType = false
	}

	// post new schedule
	res, err := c.api.UpdateSchedule(c.vin, c.pin, stat.EvInfo.Schedules)

	if err == nil && res.ResponseStatus != "pending" {
		err = fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}
	return err
}
