package fiat

import (
	"github.com/evcc-io/evcc/api"
)

type Controller struct {
	api              *API
	vin              string
	pin              string
	requestedCurrent int64
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
	var err error

	if enable {
		// Force charge start
		res, err := c.api.Action(c.vin, c.pin, "ev/chargenow", "CNOW")
		if err == nil && res.ResponseStatus == "OK" {
			// update charge schedule to start now
			// return ChangeScheduleCharge(time.Now(), nil);
		}
	} else {
		// Simulate stop charging by updating charege schedule end time
		// return ChangeScheduleCharge(nil, time.Now().add("2m"))
		err = api.ErrNotAvailable
	}

	return err
}

/*
func (c *Controller) ChangeScheduleCharge(startTime TimeMillis, endTime TimeMillis) error {
	// get current schedule
	var schedule = nil
	stat, err := c.api.Status(c.vin)
	if err != nil && stat.EvInfo != nil {
		schedule = stat.EvInfo.Schedules
	}
	if schedule == nil {
		return api.ErrNotAvailable
	}
	if endTime == nil {
		endTime = schedule[0].EndTime
	}
	if startTime == nil {
		startTime = schedule[0].StartTime
	}

	// update schedule 1 and make sure it's active
	schedule[0].CabinPriority = false
	schedule[0].ChargeToFull = false
	schedule[0].EnableScheduleType = true
	schedule[0].EndTime = endTime
	schedule[0].RepeatSchedule = true
	schedule[0].ScheduleType = "CHARGE"
	schedule[0].ScheduleDays.friday = true
	schedule[0].ScheduleDays.monday = true
	schedule[0].ScheduleDays.saturday = true
	schedule[0].ScheduleDays.sunday = true
	schedule[0].ScheduleDays.thursday = true
	schedule[0].ScheduleDays.tuesday = true
	schedule[0].ScheduleDays.wednesday = true
	schedule[0].StartTime = startTime

	// make sure the other schedules are disabled in case user changed them
	schedule[1].EnableScheduleType = false
	schedule[2].EnableScheduleType = false

	// post new schedule
	return apiError(c.api.UpdateSchedule(c.vin, c.pin, request.MarshalJSON(schedule)))
}*/
