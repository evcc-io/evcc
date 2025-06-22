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
	"github.com/jinzhu/now"
)

type Tariff struct {
	*embed
	log    *util.Logger
	data   *util.Monitor[api.Rates]
	priceG func() (float64, error)
	typ    api.TariffType
	cache  *SolarCacheManager
}

var _ api.Tariff = (*Tariff)(nil)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Tariff, error) {
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

	// Initialize cache for solar forecast tariffs
	if cc.Type == api.TariffTypeSolar && forecastG != nil {
		t.cache = NewSolarCacheManager("custom", cc)
	}

	if forecastG != nil {
		done := make(chan error)
		go t.run(forecastG, done, cc.Interval)
		err = <-done
	}

	return t, err
}

func (t *Tariff) run(forecastG func() (string, error), done chan error, interval time.Duration) {
	var once sync.Once

	// Try to load from cache on startup for solar forecasts
	if t.cache != nil {
		if cached, ok := t.cache.Get(interval); ok {
			t.log.DEBUG.Printf("loaded %d rates from cache", len(cached))
			// Apply totalPrice to cached rates if needed
			for i, r := range cached {
				cached[i] = api.Rate{
					Value: t.totalPrice(r.Value, r.Start),
					Start: r.Start.Local(),
					End:   r.End.Local(),
				}
			}
			mergeRatesAfter(t.data, cached, beginningOfDay())
			once.Do(func() { close(done) })
		}
	}

	for tick := time.Tick(interval); ; <-tick {
		var rawData api.Rates
		if err := backoff.Retry(func() error {
			s, err := forecastG()
			if err != nil {
				return backoffPermanentError(err)
			}
			if err := json.Unmarshal([]byte(s), &rawData); err != nil {
				return backoff.Permanent(err)
			}
			// Normalize timestamps to local time without applying totalPrice yet
			for i, r := range rawData {
				rawData[i] = api.Rate{
					Value: r.Value,
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

		// Cache the raw data for solar forecasts (before totalPrice adjustments)
		if t.cache != nil {
			if err := t.cache.Set(rawData); err != nil {
				t.log.DEBUG.Printf("failed to cache forecast data: %v", err)
			}
		}

		// Apply totalPrice to the data for use
		data := make(api.Rates, len(rawData))
		for i, r := range rawData {
			data[i] = api.Rate{
				Value: t.totalPrice(r.Value, r.Start),
				Start: r.Start,
				End:   r.End,
			}
		}

		// only prune rates older than current period
		periodStart := now.With(time.Now()).BeginningOfHour()
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

	res := make(api.Rates, 48)
	start := now.BeginningOfHour()

	for i := range res {
		slot := start.Add(time.Duration(i) * time.Hour)
		res[i] = api.Rate{
			Start: slot,
			End:   slot.Add(time.Hour),
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
