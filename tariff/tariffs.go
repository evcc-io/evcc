package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

type Tariffs struct {
	Currency                   currency.Unit
	Grid, FeedIn, Co2, Planner api.Tariff
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

func averagePrice(t api.Tariff) (float64, error) {
	if t != nil {
		if rr, err := t.Rates(); err == nil {
			if avg, err := rr.Average(); err == nil {
				return avg, nil
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
	return currentPrice(t.Co2)
}

func (t *Tariffs) AverageGridPrice() (float64, error) {
	return averagePrice(t.Grid)
}

func (t *Tariffs) AverageCo2() (float64, error) {
	return averagePrice(t.Co2)
}
