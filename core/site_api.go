package core

import (
	"errors"

	"github.com/andig/evcc/api"
)

// SiteAPI is the external site API
type SiteAPI interface {
	Healthy() bool
	LoadPoints() []LoadPointAPI
	SetPrioritySoC(float64) error
}

// GetPrioritySoC returns the PrioritySoC
func (site *Site) GetPrioritySoC() float64 {
	site.Lock()
	defer site.Unlock()
	return site.PrioritySoC
}

// SetPrioritySoC sets the PrioritySoC
func (site *Site) SetPrioritySoC(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if _, ok := site.batteryMeter.(api.Battery); !ok {
		return errors.New("battery not configured")
	}

	site.PrioritySoC = soc
	site.publish("prioritySoC", site.PrioritySoC)

	return nil
}
