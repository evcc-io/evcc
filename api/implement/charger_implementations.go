package implement

import (
	"github.com/evcc-io/evcc/api"
)

func CurrentController(f func(int64) error) api.CurrentController {
	return &iCurrentController{f}
}

type iCurrentController struct {
	f func(int64) error
}

func (i *iCurrentController) MaxCurrent(c int64) error {
	return i.f(c)
}

//

func CurrentGetter(f func() (float64, error)) api.CurrentGetter {
	return &iCurrentGetter{f}
}

type iCurrentGetter struct {
	f func() (float64, error)
}

func (i *iCurrentGetter) GetMaxCurrent() (float64, error) {
	return i.f()
}

//

func ChargerEx(f func(float64) error) api.ChargerEx {
	return &iChargerEx{f}
}

type iChargerEx struct {
	f func(float64) error
}

func (i *iChargerEx) MaxCurrentMillis(c float64) error {
	return i.f(c)
}

//

func PhaseSwitcher(f func(int) error) api.PhaseSwitcher {
	return &iPhaseSwitcher{f}
}

type iPhaseSwitcher struct {
	f func(int) error
}

func (i *iPhaseSwitcher) Phases1p3p(p int) error {
	return i.f(p)
}

//

func PhaseGetter(f func() (int, error)) api.PhaseGetter {
	return &iPhaseGetter{f}
}

type iPhaseGetter struct {
	f func() (int, error)
}

func (i *iPhaseGetter) GetPhases() (int, error) {
	return i.f()
}
