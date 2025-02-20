package heating

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

// PowerController controller implementation
type PowerController struct {
	phases    int
	power     int64
	maxPowerS func(int64) error
}

// NewPowerController creates power controller
func NewPowerController(ctx context.Context, maxPowerS func(int64) error, phases int) *PowerController {
	return &PowerController{
		maxPowerS: maxPowerS,
		phases:    phases,
	}
}

// MaxCurrent implements the api.Charger interface
func (wb *PowerController) MaxCurrent(current int64) error {
	return wb.MaxCurrentEx(float64(current))
}

// MaxCurrent implements the api.Charger interface
func (wb *PowerController) MaxCurrentEx(current float64) error {
	return wb.maxPower(int64(230 * current * float64(wb.phases)))
}

func (wb *PowerController) maxPower(power int64) error {
	// TODO
	if wb.maxPowerS == nil {
		return api.ErrNotAvailable
	}

	err := wb.maxPowerS(power)
	if err == nil {
		wb.power = power
	}

	return err
}
