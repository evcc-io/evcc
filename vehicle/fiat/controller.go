package fiat

import (
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
	// To avoi
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
		// Simulate stop charging by updating charege schedule end time to 1 minute in the future
		return c.ChangeScheduleCharge(c.lastStartCharge, time.Now().Add(time.Minute*1))
	}
}

func (c *Controller) ChangeScheduleCharge(startTime time.Time, endTime time.Time) error {
	// get current schedule
	var schedules []ScheduleData
	stat, err := c.api.Status(c.vin)
	if err != nil && stat.EvInfo != nil {
		schedules = stat.EvInfo.Schedules
	}
	if schedules == nil {
		return api.ErrNotAvailable
	}

	// update schedule 1 and make sure it's active
	// all values are set to be sure no manual change can lead to inconsistencies
	schedules[0].CabinPriority = false
	schedules[0].ChargeToFull = false
	schedules[0].EnableScheduleType = true
	schedules[0].EndTime = endTime.Format("hh:mm")
	schedules[0].RepeatSchedule = true
	schedules[0].ScheduleType = "CHARGE"
	schedules[0].ScheduledDays.Friday = true
	schedules[0].ScheduledDays.Monday = true
	schedules[0].ScheduledDays.Saturday = true
	schedules[0].ScheduledDays.Sunday = true
	schedules[0].ScheduledDays.Thursday = true
	schedules[0].ScheduledDays.Tuesday = true
	schedules[0].ScheduledDays.Wednesday = true
	schedules[0].StartTime = startTime.Format("hh:mm")

	// make sure the other schedules are disabled in case user changed them
	schedules[1].EnableScheduleType = false
	schedules[2].EnableScheduleType = false

	// post new schedule
	res, err := c.api.UpdateSchedule(c.vin, c.pin, schedules)

	if err == nil && res.ResponseStatus != "200" {
		err = api.ErrMustRetry
	}

	return err
}
