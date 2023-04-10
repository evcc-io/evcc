package core

import "github.com/evcc-io/evcc/api"

type currentController struct {
	charger api.CurrentController
}

var _ api.PowerController = (*currentController)(nil)

// currentController implements the api.PowerController interface
func (c *currentController) MaxPower(power float64) error {
	return api.ErrNotAvailable
}
