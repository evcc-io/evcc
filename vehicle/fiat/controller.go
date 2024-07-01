package fiat

import (
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Controller struct {
	api *API,
	pin string
}

// NewController creates a vehicle current and charge controller
func NewController(api *API, pin string) *Controller {
	impl := &Controller{
		api: api,
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
		err = apiError(c.api.action("ev/chargenow"))
		if err != nil {
			if slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
				return nil
			} else {
				return err
			}
		}
		// TODO: update schedule
		 
	} else {
		// TODO: simulate stop charging by updating schedule
		err = api.ErrNotAvailable
	}

	return err
}
