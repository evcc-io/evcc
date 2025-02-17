package tariff

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

type Tariffs struct {
	Currency                          currency.Unit
	Grid, FeedIn, Co2, Planner, Solar api.Tariff
}

func Now(t api.Tariff) (float64, error) {
	if t != nil {
		if rr, err := t.Rates(); err == nil {
			if r, err := rr.Current(time.Now()); err == nil {
				return r.Price, nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}

func Forecast(t api.Tariff) api.Rates {
	staticTariffs := []api.TariffType{api.TariffTypePriceStatic, api.TariffTypePriceDynamic}
	if t != nil && !slices.Contains(staticTariffs, t.Type()) {
		if rr, err := t.Rates(); err == nil {
			return rr
		}
	}
	return nil
}

func (t *Tariffs) Get(u api.TariffUsage) api.Tariff {
	switch u {
	case api.TariffUsageCo2:
		return t.Co2

	case api.TariffUsageFeedIn:
		return t.FeedIn

	case api.TariffUsageGrid:
		return t.Grid

	// TODO solar
	case api.TariffUsagePlanner:
		switch {
		case t.Planner != nil:
			// prio 0: manually set planner tariff
			return t.Planner

		case t.Grid != nil && t.Grid.Type() == api.TariffTypePriceForecast:
			// prio 1: grid tariff with forecast
			return t.Grid

		case t.Co2 != nil:
			// prio 2: co2 tariff
			return t.Co2

		default:
			// prio 3: static grid tariff
			return t.Grid
		}

	case api.TariffUsageSolar:
		return t.Solar

	default:
		return nil
	}
}
