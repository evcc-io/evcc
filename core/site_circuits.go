package core

import (
	"errors"
	"fmt"
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

type circuitStruct struct {
	Title      string   `json:"title,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	Power      float64  `json:"power"`
	Current    *float64 `json:"current,omitempty"`
	MaxPower   float64  `json:"maxPower,omitempty"`
	MaxCurrent float64  `json:"maxCurrent,omitempty"`
	Dimmed     bool     `json:"dimmed"`
}

// publishCircuits returns a list of circuit titles
func (site *Site) publishCircuits() {
	cc := config.Circuits().Devices()
	res := make(map[string]circuitStruct, len(cc))

	for _, c := range cc {
		instance := c.Instance()
		props := deviceProperties(c)

		data := circuitStruct{
			Title:      props.Title,
			Icon:       props.Icon,
			Power:      instance.GetChargePower(),
			MaxPower:   instance.GetMaxPower(),
			MaxCurrent: instance.GetMaxCurrent(),
			Dimmed:     instance.Dimmed(),
		}

		if instance.GetMaxCurrent() > 0 {
			data.Current = lo.EmptyableToPtr(instance.GetMaxPhaseCurrent())
		}

		res[c.Config().Name] = data
	}

	site.publish(keys.Circuits, res)
}

func (site *Site) dimMeters(dim bool) error {
	var errs error

	for _, dev := range slices.Concat(site.auxMeters, site.extMeters) {
		m, ok := dev.Instance().(api.Dimmer)
		if !ok {
			continue
		}

		if dimmed, err := m.Dimmed(); err == nil {
			if dim == dimmed {
				continue
			}
		} else {
			if !errors.Is(err, api.ErrNotAvailable) {
				errs = errors.Join(errs, fmt.Errorf("%s dimmed: %w", dev.Config().Name, err))
			}
			continue
		}

		if err := m.Dim(dim); err == nil {
			site.log.DEBUG.Printf("%s dim: %t", dev.Config().Name, dim)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dim: %w", dev.Config().Name, err))
		}
	}

	return errs
}
