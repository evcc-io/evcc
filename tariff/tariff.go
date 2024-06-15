package tariff

import (
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
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

func NewConfigurableFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed    `mapstructure:",squash"`
		Price    *provider.Config
		Forecast *provider.Config
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
		priceG, err = provider.NewFloatGetterFromConfig(*cc.Price)
		if err != nil {
			return nil, fmt.Errorf("price: %w", err)
		}
	}

	if cc.Forecast != nil {
		forecastG, err = provider.NewStringGetterFromConfig(*cc.Forecast)
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
	bo := newBackoff()

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
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data.Sort()

		t.data.Set(data)
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
