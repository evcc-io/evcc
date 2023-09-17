package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type currentAdapter struct {
	api.CurrentController
	log                    *util.Logger
	minCurrent, maxCurrent float64
}

var _ api.PowerLimiter = (*currentAdapter)(nil)

// currentAdapter implements the api.PowerLimiter interface
func (c *currentAdapter) MaxPower(power float64) (float64, error) {
	return 0, api.ErrNotAvailable
}
