package tariff

import (
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/evcc-io/evcc/util"
)

type Fixed struct {
	clock   clock.Clock
	zones   fixed.Zones
	dynamic bool
}

var _ api.Tariff = (*Fixed)(nil)

func init() {
	registry.Add("fixed", NewFixedFromConfig)
}

func NewFixedFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Price float64
		Zones fixed.ZoneConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	zones, err := cc.Zones.Parse(cc.Price)
	if err != nil {
		return nil, err
	}

	t := &Fixed{
		clock:   clock.New(),
		dynamic: len(cc.Zones) >= 1,
		zones:   zones,
	}

	return t, nil
}

// Rates implements the api.Tariff interface
func (t *Fixed) Rates() (api.Rates, error) {
	return t.zones.Rates(t.clock.Now())
}

// Type implements the api.Tariff interface
func (t *Fixed) Type() api.TariffType {
	if t.dynamic {
		return api.TariffTypePriceForecast
	}
	return api.TariffTypePriceStatic
}
