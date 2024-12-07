package tariff

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/nordpool"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Nordpool struct {
	*embed
	log      *util.Logger
	area     string
	currency string
	data     *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Nordpool)(nil)

func init() {
	registry.Add("nordpool", NewNordpoolFromConfig)
}

func NewNordpoolFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed    `mapstructure:",squash"`
		Area     string
		Currency string
	}

	err := util.DecodeOther(other, &cc)
	if err != nil {
		return nil, err
	}

	if cc.Area == "" {
		return nil, errors.New("missing area")
	}

	if cc.Currency == "" {
		return nil, errors.New("missing currency")
	}

	err = cc.init()
	if err != nil {
		return nil, err
	}

	t := &Nordpool{
		embed:    &cc.embed,
		log:      util.NewLogger("nordpool"),
		area:     strings.ToUpper(cc.Area),
		currency: strings.ToUpper(cc.Currency),
		data:     util.NewMonitor[api.Rates](2 * time.Hour),
	}
	done := make(chan error)

	t.log.DEBUG.Printf("downloading price data")
	go t.run(done)
	err = <-done
	return t, err
}

func (t *Nordpool) run(done chan error) {
	var once sync.Once
	var n nordpool.NordpoolResponse
	client := request.NewHelper(t.log)

	for tick := time.Tick(time.Hour); true; <-tick {
		rd := time.Now().Local()
		uri := nordpool.MakeURL(t.area, rd, t.currency)

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &n))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		rates := make(api.Rates, 0, 48)
		for _, ae := range n.AreaEntries {
			r := api.Rate{
				Start: ae.Start.Local(),
				End:   ae.Stop.Local(),
				Price: t.totalPrice(float64(ae.Entry[t.area])/1000, ae.Start),
			}
			rates = append(rates, r)
		}

		uri = nordpool.MakeURL(t.area, time.Now().AddDate(0, 0, 1), t.currency)

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &n))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		for _, ae := range n.AreaEntries {
			r := api.Rate{
				Start: ae.Start.Local(),
				End:   ae.Stop.Local(),
				Price: t.totalPrice(float64(ae.Entry[t.area])/1000, ae.Start),
			}

			rates = append(rates, r)
		}

		mergeRates(t.data, rates)
		once.Do(func() { close(done) })
	}
}

func (t *Nordpool) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *Nordpool) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
