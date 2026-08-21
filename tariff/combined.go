package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

type combined struct {
	tariffs []api.Tariff
}

func NewCombined(tariffs []api.Tariff) api.Tariff {
	return &combined{
		tariffs: tariffs,
	}
}

func (t *combined) Rates() (api.Rates, error) {
	var rates api.Rates
	var err error
	var errs int

	for _, t := range t.tariffs {
		rr, rerr := t.Rates()
		if rerr != nil {
			err = rerr
			errs++
			continue
		}

		rates = append(rates, rr...)
	}

	// only fail if all sources are unavailable - a single stale/erroring
	// source should not discard the other sources' still-valid rates
	if errs == len(t.tariffs) {
		return nil, err
	}

	rates.Sort()

	var res api.Rates

	partitions := lo.PartitionBy(rates, func(r api.Rate) time.Time {
		return r.Start
	})

	for _, rr := range partitions {
		res = append(res, api.Rate{
			Start: rr[0].Start,
			End:   rr[0].End,
			Value: lo.SumBy(rr, func(r api.Rate) float64 {
				return r.Value
			}),
		})
	}

	return res, nil
}

func (t *combined) Type() api.TariffType {
	return api.TariffTypeSolar
}
