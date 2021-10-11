package tariff

import (
	"github.com/evcc-io/evcc/api"
)

type Tariffs struct {
	Grid   api.Tariff
	Feedin api.Tariff
}

var _ api.Tariff = (*Fixed)(nil)

func NewTariffs(grid api.Tariff, feedin api.Tariff) *Tariffs {
	t := Tariffs{}
	t.Grid = grid
	t.Feedin = feedin
	return &t
}
