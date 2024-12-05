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
	t.log.TRACE.Printf("nordpool type: %#v\n", t)
	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		rd := time.Now().Local()
		url, _ := nordpool.MakeURL(t.area, rd, t.currency)

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(url.String(), &n))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}
		t.log.TRACE.Printf("Returned from url reader with n: %v\n", n)
		rates := make(api.Rates, 0, len(n.AreaEntries))
		for _, ae := range n.AreaEntries {
			r := api.Rate{
				Start: ae.Start.Local(),
				End:   ae.Stop.Local(),
				Price: t.totalPrice(float64(ae.Entry[t.area])/1000, ae.Start),
			}
			t.log.TRACE.Printf("api.Rate: %v\n", r)
			rates = append(rates, r)
		}
		t.log.TRACE.Printf("Merging %v with %v\n", t.data, rates)
		mergeRates(t.data, rates)

		rd = time.Now().AddDate(0, 0, 1)
		url, _ = nordpool.MakeURL(t.area, rd, t.currency)

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(url.String(), &n))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}
		t.log.TRACE.Printf("Returned from url reader with n: &v\n", n)
		rates = make(api.Rates, 0, len(n.AreaEntries))
		for _, ae := range n.AreaEntries {
			r := api.Rate{
				Start: ae.Start.Local(),
				End:   ae.Stop.Local(),
				Price: t.totalPrice(float64(ae.Entry[t.area])/1000, ae.Start),
			}
			t.log.TRACE.Printf("api.Rate: %v\n", r)
			rates = append(rates, r)
		}
		t.log.TRACE.Printf("Merging %v with %v\n", t.data, rates)
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
