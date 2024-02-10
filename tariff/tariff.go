package tariff

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
)

type Tariff struct {
	*embed
	priceG func() (float64, error)
}

var _ api.Tariff = (*Tariff)(nil)

func init() {
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

func NewConfigurableFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed `mapstructure:",squash"`
		Price provider.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	priceG, err := provider.NewFloatGetterFromConfig(cc.Price)
	if err != nil {
		return nil, fmt.Errorf("price: %w", err)
	}

	t := &Tariff{
		embed:  &cc.embed,
		priceG: priceG,
	}

	return t, nil
}

// Rates implements the api.Tariff interface
func (t *Tariff) Rates() (api.Rates, error) {
	price, err := t.priceG()
	if err != nil {
		return nil, err
	}

	res := make(api.Rates, 48)
	start := now.BeginningOfHour()

	for i := 0; i < len(res); i++ {
		slot := start.Add(time.Duration(i) * time.Hour)
		res[i] = api.Rate{
			Start: slot,
			End:   slot.Add(time.Hour),
			Price: t.totalPrice(price),
		}
	}

	return res, nil
}

// Type implements the api.Tariff interface
func (t *Tariff) Type() api.TariffType {
	return api.TariffTypePriceDynamic
}
