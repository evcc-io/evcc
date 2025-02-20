package heating

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

// PowerModeController controller implementation
type PowerModeController struct {
	phases    int
	power     int64
	maxPowerG func() (int64, error)
	maxPowerS func(int64) error
}

// NewPowerModeController creates power mode controller
func NewPowerModeController(ctx context.Context, maxPowerS func(int64) error, maxPowerG func() (int64, error), phases int) *PowerModeController {
	return &PowerModeController{
		maxPowerG: maxPowerG,
		maxPowerS: maxPowerS,
		phases:    phases,
	}
}

func (wb *PowerModeController) getMaxPower() (int64, error) {
	if wb.maxPowerG == nil {
		return wb.power, nil
	}
	return wb.maxPowerG()
}

// Status implements the api.Charger interface
func (wb *PowerModeController) Status() (api.ChargeStatus, error) {
	power, err := wb.getMaxPower()
	if err != nil {
		return api.StatusNone, err
	}

	status := map[bool]api.ChargeStatus{false: api.StatusB, true: api.StatusC}
	return status[power > 0], nil
}

// Enabled implements the api.Charger interface
func (wb *PowerModeController) Enabled() (bool, error) {
	power, err := wb.getMaxPower()
	return power > 0, err
}

// Enable implements the api.Charger interface
func (wb *PowerModeController) Enable(enable bool) error {
	var power int64
	if enable {
		power = wb.power
	}
	return wb.setMaxPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *PowerModeController) MaxCurrent(current int64) error {
	return wb.MaxCurrentEx(float64(current))
}

// MaxCurrent implements the api.Charger interface
func (wb *PowerModeController) MaxCurrentEx(current float64) error {
	return wb.setMaxPower(int64(230 * current * float64(wb.phases)))
}

func (wb *PowerModeController) setMaxPower(power int64) error {
	err := wb.maxPowerS(power)
	if err == nil {
		wb.power = power
	}

	return err
}
