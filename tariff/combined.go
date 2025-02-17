package tariff

import (
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

type combined struct {
	tt []api.Tariff
}

func NewCombined(tt []api.Tariff) api.Tariff {
	return &combined{
		tt: tt,
	}
}

func (t *combined) Rates() (api.Rates, error) {
	var keys []time.Time
	for _, t := range t.tt {
		rr, err := t.Rates()
		if err != nil {
			return nil, err
		}

		for _, r := range rr {
			keys = append(keys, r.Start)
		}
	}

	keys = slices.SortedFunc(slices.Values(lo.Uniq(keys)), func(a, b time.Time) int {
		return a.Compare(b)
	})

	var res api.Rates
	for _, ts := range keys {
		var rate api.Rate

		for _, t := range t.tt {
			r, err := At(t, ts)
			if err != nil {
				continue
			}

			if rate.Start.IsZero() {
				rate = r
				continue
			}

			if !r.End.Equal(rate.End) {
				return nil, errors.New("combined tariffs must have the same period length")
			}

			rate.Price += r.Price
		}

		res = append(res, rate)
	}

	return res, nil
}

func (t *combined) Type() api.TariffType {
	return api.TariffTypeSolar
}
