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
	// t tariff is base cost
	trs, err := t.t.Rates()
	if err != nil {
		return nil, err
	}

	// g tariff is generation
	grs, err := t.g.Rates()
	if err != nil {
		return nil, err
	}

	res := make(api.Rates, 0, len(trs))

	for _, tr := range trs {
		if gr, err := grs.Current(tr.Start); err == nil {
			// adjust price
			if gr.Price >= t.maxPower {
				tr.Price = 0
			} else {
				// fmt.Printf("%.1f * 1-(%.1f/%.1f)\n", tr.Price, gr.Price, t.maxPower)
				tr.Price *= 1 - (gr.Price / t.maxPower)
			}
		}

		res = append(res, tr)
	}

	return res, nil
}

func (t *adjusted) Type() api.TariffType {
	return t.t.Type()
}
