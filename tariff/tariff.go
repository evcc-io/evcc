package tariff

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
)

type Tariff struct {
	*embed
	log    *util.Logger
	data   *util.Monitor[api.Rates]
	priceG func() (float64, error)
}

var _ api.Tariff = (*Tariff)(nil)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		Price    *provider.Config
		Forecast *provider.Config
		Cache    time.Duration
	}{
		Cache: 15 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if (cc.Price != nil) == (cc.Forecast != nil) {
		return nil, fmt.Errorf("must have either price or forecast")
	}

	var (
		err       error
		priceG    func() (float64, error)
		forecastG func() (string, error)
	)

	if cc.Price != nil {
		priceG, err = provider.NewFloatGetterFromConfig(ctx, *cc.Price)
		if err != nil {
			return nil, fmt.Errorf("price: %w", err)
		}

		priceG = provider.Cached(priceG, cc.Cache)
	}

	if cc.Forecast != nil {
		forecastG, err = provider.NewStringGetterFromConfig(ctx, *cc.Forecast)
		if err != nil {
			return nil, fmt.Errorf("forecast: %w", err)
		}
	}

	t := &Tariff{
		log:    util.NewLogger("tariff"),
		embed:  &cc.embed,
		priceG: priceG,
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	if forecastG != nil {
		done := make(chan error)
		go t.run(forecastG, done)
		err = <-done
	}

	return t, err
}

func (t *Tariff) run(forecastG func() (string, error), done chan error) {
	var once sync.Once

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var data api.Rates
		if err := backoff.Retry(func() error {
			s, err := forecastG()
			if err != nil {
				return backoffPermanentError(err)
			}
			if err := json.Unmarshal([]byte(s), &data); err != nil {
				return backoff.Permanent(err)
			}
			for i, r := range data {
				data[i].Price = t.totalPrice(r.Price)
			}
			return nil
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

func (t *Tariff) forecastRates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *Tariff) priceRates() (api.Rates, error) {
	price, err := t.priceG()
	if err != nil {
		return nil, err
	}

	res := make(api.Rates, 48)
	start := now.BeginningOfHour()

	for i := range res {
		slot := start.Add(time.Duration(i) * time.Hour)
		res[i] = api.Rate{
			Start: slot,
			End:   slot.Add(time.Hour),
			Price: t.totalPrice(price),
		}
	}

	return res, nil
}

// Rates implements the api.Tariff interface
func (t *Tariff) Rates() (api.Rates, error) {
	if t.priceG != nil {
		return t.priceRates()
	}

	return t.forecastRates()
}

// Type implements the api.Tariff interface
func (t *Tariff) Type() api.TariffType {
	if t.priceG != nil {
		return api.TariffTypePriceDynamic
	}
	return api.TariffTypePriceForecast
}
