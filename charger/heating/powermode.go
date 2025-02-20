package heating

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

// PowerModeController controller implementation
type PowerModeController struct {
	ctrl      *PowerController
	maxPowerG func() (int64, error)
}

// NewPowerModeController creates power mode controller
func NewPowerModeController(ctx context.Context, ctrl *PowerController, maxPowerG func() (int64, error)) *PowerModeController {
	return &PowerModeController{
		ctrl:      ctrl,
		maxPowerG: maxPowerG,
	}
}

func (wb *PowerModeController) power() (int64, error) {
	if wb.maxPowerG == nil {
		return wb.ctrl.power, nil
	}
	return wb.maxPowerG()
}

// Status implements the api.Charger interface
func (wb *PowerModeController) Status() (api.ChargeStatus, error) {
	power, err := wb.power()
	if err != nil {
		return api.StatusNone, err
	}

	status := map[bool]api.ChargeStatus{false: api.StatusB, true: api.StatusC}
	return status[power > 0], nil
}

// Enabled implements the api.Charger interface
func (wb *PowerModeController) Enabled() (bool, error) {
	power, err := wb.power()
	return power > 0, err
}

// Enable implements the api.Charger interface
func (wb *PowerModeController) Enable(enable bool) error {
	var power int64
	if enable {
		power = wb.ctrl.power
	}
	return wb.ctrl.maxPower(power)
}
