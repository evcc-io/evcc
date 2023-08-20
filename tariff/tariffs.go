package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

const (
	Grid       = "grid"
	Feedin     = "feedin"
	Generation = "generation"
	Planner    = "planner"
)

type Tariffs struct {
	Currency                               currency.Unit
	Grid, FeedIn, Co2, Generation, Planner api.Tariff
}

func NewTariffs(currency currency.Unit, grid, feedin, co2, generation, planner api.Tariff) *Tariffs {
	return &Tariffs{
		Currency:   currency,
		Grid:       grid,
		FeedIn:     feedin,
		Co2:        co2,
		Generation: generation,
		Planner:    planner,
	}
}

// Get returns the respective tariff if configured or nil
func (t *Tariffs) Get(tariff string) api.Tariff {
	switch tariff {
	case Grid:
		return t.Grid

	case Feedin:
		return t.FeedIn

	case Generation:
		return t.Generation

	case Planner:
		switch {
		case t.Planner != nil:
			// prio 0: manually set planner tariff
			return t.Planner

		case t.Grid != nil && t.Grid.Type() == api.TariffTypePriceDynamic:
			// prio 1: dynamic grid tariff
			return t.Grid

		case t.Co2 != nil:
			// prio 2: co2 tariff
			return t.Co2

		default:
			// prio 3: static grid tariff
			return t.Grid
		}

	default:
		return nil
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
	if t.Co2 != nil {
		return currentPrice(t.Co2)
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
