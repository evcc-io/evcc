package tariff

import (
	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

type Tariffs struct {
	Currency currency.Unit
	Grid     api.Tariff
	FeedIn   api.Tariff
}

func NewTariffs(currency currency.Unit, grid api.Tariff, feedin api.Tariff) *Tariffs {
	t := Tariffs{}
	t.Currency = currency
	t.Grid = grid
	t.FeedIn = feedin
	return &t
}
