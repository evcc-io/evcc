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

	for _, t := range t.tariffs {
		rr, err := t.Rates()
		if err != nil {
			return nil, err
		}

		rates = append(rates, rr...)
	}

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
