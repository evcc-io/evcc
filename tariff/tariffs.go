package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

type Tariffs struct {
	Currency              currency.Unit
	Grid, FeedIn, Planner api.Tariff
}

func NewTariffs(currency currency.Unit, grid, feedin, planner api.Tariff) *Tariffs {
	if planner == nil {
		planner = grid
	}

	return &Tariffs{
		Currency: currency,
		Grid:     grid,
		FeedIn:   feedin,
		Planner:  planner,
	}
}

func currentPrice(t api.Tariff) (float64, error) {
	if t != nil {
		if rr, err := t.Rates(); err == nil {
			if r, err := rr.Current(time.Now()); err == nil {
				return r.Price, nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}

// CurrentGridPrice returns the current feedin price.
func (t *Tariffs) CurrentGridPrice() (float64, error) {
	return currentPrice(t.Grid)
}

// CurrentFeedInPrice returns the current feedin price.
func (t *Tariffs) CurrentFeedInPrice() (float64, error) {
	return currentPrice(t.FeedIn)
}

// CurrentEffectivePrice calculates the real energy price based on self-produced and grid-imported energy.
func (t *Tariffs) CurrentEffectivePrice(greenShare float64) (float64, error) {
	if grid, err := t.CurrentGridPrice(); err == nil {
		feedin, err := t.CurrentFeedInPrice()
		if err != nil {
			feedin = 0
		}
		return grid*(1-greenShare) + feedin*greenShare, nil
	}
	return 0, api.ErrNotAvailable
}

// CurrentCo2 determins the grids co2 emission.
func (t *Tariffs) CurrentCo2() (float64, error) {
	if t.Planner != nil && t.Planner.Unit() == "gCO2eq" {
		return currentPrice(t.Planner)
	}
	return 0, api.ErrNotAvailable
}

// CurrentEffectiveCo2 calculates the amount of emitted co2 based on self-produced and grid-imported energy.
func (t *Tariffs) CurrentEffectiveCo2(greenShare float64) (float64, error) {
	if co2, err := t.CurrentCo2(); err == nil {
		return co2 * (1 - greenShare), nil
	}
	return 0, api.ErrNotAvailable
}

// outdatedError returns api.ErrOutdated if t is older than 2*d
func outdatedError(t time.Time, d time.Duration) error {
	if time.Since(t) > 2*d {
		return api.ErrOutdated
	}
	return nil
}
