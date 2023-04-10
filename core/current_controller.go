package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type currentLimiter struct {
	log                    *util.Logger
	charger                api.CurrentLimiter
	minCurrent, maxCurrent float64
}

var _ api.PowerLimiter = (*currentLimiter)(nil)

// currentLimiter implements the api.PowerLimiter interface
func (c *currentLimiter) MaxPower(power float64) (float64, error) {
	return 0, api.ErrNotAvailable
}
