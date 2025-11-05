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
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

type Tariff struct {
	*embed
	log    *util.Logger
	data   *util.Monitor[api.Rates]
	priceG func() (float64, error)
	typ    api.TariffType
}

var _ api.Tariff = (*Tariff)(nil)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		Price    *plugin.Config
		Forecast *plugin.Config
		Type     api.TariffType `mapstructure:"tariff"`
		Interval time.Duration
		Cache    time.Duration
	}{
		Interval: time.Hour,
		Cache:    15 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if (cc.Price != nil) == (cc.Forecast != nil) {
		return nil, fmt.Errorf("must have either price or forecast")
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	priceG, err := cc.Price.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("price: %w", err)
	}
	if priceG != nil {
		priceG = util.Cached(priceG, cc.Cache)
	}

	forecastG, err := cc.Forecast.StringGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("forecast: %w", err)
	}

	t := &Tariff{
		log:    util.NewLogger("tariff"),
		embed:  &cc.embed,
		typ:    cc.Type,
		priceG: priceG,
		data:   util.NewMonitor[api.Rates](2 * cc.Interval),
	}

	if forecastG != nil {
		done := make(chan error)
		go t.run(forecastG, done, cc.Interval)

		if err := <-done; err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *Tariff) run(forecastG func() (string, error), done chan error, interval time.Duration) {
	var once sync.Once

	for tick := time.Tick(interval); ; <-tick {
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
				data[i] = api.Rate{
					Value: t.totalPrice(r.Value, r.Start),
					Start: r.Start.Local(),
					End:   r.End.Local(),
				}
			}
			return nil
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		// only prune rates older than current period
		periodStart := time.Now().Truncate(SlotDuration)
		if t.typ == api.TariffTypeSolar {
			periodStart = beginningOfDay()
		}
		mergeRatesAfter(t.data, data, periodStart)

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

	res := make(api.Rates, 48*4) // forecast for two days
	start := time.Now().Truncate(SlotDuration)

	for i := range res {
		slot := start.Add(time.Duration(i) * SlotDuration)
		res[i] = api.Rate{
			Start: slot,
			End:   slot.Add(SlotDuration),
			Value: t.totalPrice(price, slot),
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
	switch {
	case t.typ != 0:
		return t.typ
	case t.priceG != nil:
		return api.TariffTypePriceDynamic
	default:
		return api.TariffTypePriceForecast
	}
}
