package tariff

import (
	"github.com/evcc-io/evcc/api"
)

type adjusted struct {
	t, g     api.Tariff
	maxPower float64
}

func NewAdjusted(t, g api.Tariff, maxPower float64) api.Tariff {
	return &adjusted{t, g, maxPower}
}

func (t *adjusted) Rates() (api.Rates, error) {
	trs, err := t.t.Rates()
	if err != nil {
		return nil, err
	}

	grs, err := t.g.Rates()
	if err != nil {
		return nil, err
	}

	res := make(api.Rates, 0, len(trs))

	for _, tr := range trs {
		gr, err := grs.Current(tr.Start)
		if err != nil {
			continue
		}

		if gr.Price >= t.maxPower {
			tr.Price = 0
		} else {
			tr.Price *= 1 - (gr.Price / t.maxPower)
		}

		res = append(res, tr)
	}

	return res, nil
}

func (t *adjusted) Type() api.TariffType {
	return t.t.Type()
}
