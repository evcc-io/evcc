package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decoratePeblar(base *Peblar, phaseSwitcher func(int) error) api.Charger {
	switch {
	case phaseSwitcher == nil:
		return base

	case phaseSwitcher != nil:
		return &struct {
			*Peblar
			api.PhaseSwitcher
		}{
			Peblar: base,
			PhaseSwitcher: &decoratePeblarPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}
	}

	return nil
}

type decoratePeblarPhaseSwitcherImpl struct {
	phaseSwitcher func(int) error
}

func (impl *decoratePeblarPhaseSwitcherImpl) Phases1p3p(p0 int) error {
	return impl.phaseSwitcher(p0)
}
