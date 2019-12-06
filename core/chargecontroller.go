package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// ChargeController is an api.ChargeController implementation with
// configurable getters and setters.
type ChargeController struct {
	maxCurrentS provider.IntSetter
}

// NewChargeController creates a new charge controller
func NewChargeController(maxCurrentS provider.IntSetter) api.ChargeController {
	return &ChargeController{
		maxCurrentS: maxCurrentS,
	}
}

// MaxCurrent implements the ChargeController.MaxCurrent API
func (m *ChargeController) MaxCurrent(current int64) error {
	return m.maxCurrentS(current)
}
