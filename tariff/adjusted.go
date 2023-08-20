package tariff

import (
	"github.com/evcc-io/evcc/api"
)

type adjusted struct {
	t, g, f  api.Tariff
	maxPower float64
}

func NewAdjusted(t, g, f api.Tariff, maxPower float64) api.Tariff {
	return &adjusted{t, g, f, maxPower}
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

	// f tariff is feedin
	var frs api.Rates
	if t.f != nil {
		if frs, err = t.g.Rates(); err != nil {
			return nil, err
		}
	}

	res := make(api.Rates, 0, len(trs))

	for _, tr := range trs {
		if gr, err := grs.Current(tr.Start); err == nil {
			// coverage is the fraction of maxPower that is covered by generation, capped at 100%
			coverage := min(gr.Price/t.maxPower, 1)

			switch {
			case frs == nil:
				// adjust co2
				tr.Price *= 1 - coverage

			case frs != nil:
				// adjust price to feedin tariff (entgangene Einspeisung)
				fr, err := frs.Current(tr.Start)

				if err == nil {
					tr.Price = tr.Price*(1-coverage) + fr.Price*coverage
				} else {
					tr.Price *= 1 - coverage
				}
			}
		}

		res = append(res, tr)
	}

	return res, nil
}

func (t *adjusted) Type() api.TariffType {
	return t.t.Type()
}
