package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type currentController struct {
	log                    *util.Logger
	charger                api.CurrentController
	minCurrent, maxCurrent float64
}

var _ api.PowerController = (*currentController)(nil)

// currentController implements the api.PowerController interface
func (c *currentController) MaxPower(power float64) error {
	return api.ErrNotAvailable
}
