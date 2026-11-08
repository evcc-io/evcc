package tesla

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
)

const ProxyBaseUrl = "https://api.myteslamate.com"

// Commander sends charge commands to the vehicle, via proxy or signed in-process
type Commander interface {
	api.CurrentController
	api.ChargeController
}

type Controller struct {
	api *API
	vin string
}

// NewController creates a vehicle current and charge controller
func NewController(api *API, vin string) *Controller {
	v := &Controller{
		api: api,
		vin: vin,
	}

	return v
}

var _ Commander = (*Controller)(nil)

// MaxCurrent implements the api.CurrentController interface
func (v *Controller) MaxCurrent(current int64) error {
	return apiError(v.api.Command(v.vin, "set_charging_amps", map[string]int64{"charging_amps": current}))
}

// ChargeEnable implements the api.ChargeController interface
func (v *Controller) ChargeEnable(enable bool) error {
	var err error

	if enable {
		err = apiError(v.api.Command(v.vin, "charge_start", nil))
		// proxy wraps these, e.g. "car could not execute command: complete"
		if err != nil && (strings.HasSuffix(err.Error(), "complete") || strings.HasSuffix(err.Error(), "is_charging")) {
			return nil
		}
	} else {
		err = apiError(v.api.Command(v.vin, "charge_stop", nil))

		// ignore sleeping vehicle
		if errors.Is(err, api.ErrAsleep) {
			err = nil
		}
	}

	return err
}
