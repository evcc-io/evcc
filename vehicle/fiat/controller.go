package fiat

import (
	"errors"
	"slices"

	"github.com/evcc-io/evcc/api"
)

type Controller struct {
	api *API,
	vin	string,
	pin string
}

// NewController creates a vehicle current and charge controller
func NewController(api *API, vin string, pin string) *Controller {
	impl := &Controller{
		api: api,
		vin: vin,
		pin: pin
	}
	return impl
}
var _ api.ChargeController = (*Controller)(nil)

// ChargeEnable implements the api.ChargeController interface
func (c *Controller) ChargeEnable(enable bool) error {
	if c.pin == "" {
		return api.ErrNotAuthorized
	}
	var err error

	if enable {
		// Force charge start
		err = apiError(c.api.Action(c.vin, c.pin, "ev/chargenow", "CNOW"))
		if err != nil {
			if slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
				return nil
			} else {
				return err
			}
		}
		// update charge schedule to start now 
		// return ChangeScheduleCharge(time.Now(), nil);
		 
	} else {
		// Simulate stop charging by updating charege schedule end time 
		// return ChangeScheduleCharge(nil, time.Now().add("2m"))
		err = api.ErrVehicleNotAvailable
	}

	return err
}

func (c *Controller) ChangeScheduleCharge(startTime Time, endTime Time) error {
	// get current schedule
	var schedule
	stat, err := c.api.Status(c.vin)
	if err != nil && stat.EvInfo != nil {
		schedule = stat.EvInfo.Schedules
	}
	if schedule == nil {
		return api.ErrVehicleNotAvailable
	}
	if endTime == nil {
		endTime = schedule[0].EndTime
	}
	if startTime == nil {
		startTime = schedule[0].StartTime
	}

	// update schedule 1 and make sure it's active
	schedule[0].CabinPriority= false
	schedule[0].ChargeToFull= false
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
}

