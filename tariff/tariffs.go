package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

// Co2Equivalent is the unit for co2 emissions
const Co2Equivalent = "gCO2eq"

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

// CurrentGridPrice returns the current grid price.
func (t *Tariffs) CurrentGridPrice() (float64, error) {
	return currentPrice(t.Grid)
}

// CurrentFeedInPrice returns the current feed-in price.
func (t *Tariffs) CurrentFeedInPrice() (float64, error) {
	return currentPrice(t.FeedIn)
}

// CurrentCo2 determines the grids co2 emission.
func (t *Tariffs) CurrentCo2() (float64, error) {
	if t.Planner != nil && t.Planner.Unit() == Co2Equivalent {
		return currentPrice(t.Planner)
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
