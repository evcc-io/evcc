package tariff

import (
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
