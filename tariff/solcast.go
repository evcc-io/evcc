package tariff

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/solcast"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jinzhu/now"
)

type Solcast struct {
	*request.Helper
	log   *util.Logger
	sites []string
	data  *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Solcast)(nil)

func init() {
	registry.Add("solcast", NewSolcastFromConfig)
}

func NewSolcastFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Site  string
		Token string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Site == "" {
		return nil, errors.New("missing site id")
	}
	// TODO multiple sites
	// if len(cc.Site) > 1 {
	// 	return nil, errors.New("multiple sites not supported (yet)")
	// }

	if cc.Token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("solcast").Redact(cc.Token)

	t := &Solcast{
		log:    log,
		sites:  []string{cc.Site},
		Helper: request.NewHelper(log),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = transport.BearerAuth(cc.Token, t.Client.Transport)

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Solcast) run(done chan error) {
	var once sync.Once

	// don't exceed 10 requests per 24h
	for ; true; <-time.Tick(time.Duration(3*len(t.sites)) * time.Hour) {
		var res solcast.Forecasts

		if err := backoff.Retry(func() error {
			for _, site := range t.sites {
				uri := fmt.Sprintf("https://api.solcast.com.au/rooftop_sites/%s/forecasts?period=PT60M&format=json", site)
				if err := t.GetJSON(uri, &res); err != nil {
					return err
				}
			}
			return nil
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		data := make(api.Rates, 0, len(res.Forecasts))

	NEXT:
		for _, r := range res.Forecasts {
			start := now.With(r.PeriodEnd.Add(-r.Period.Duration())).BeginningOfHour().Local()
			rr := api.Rate{
				Start: start,
				End:   start.Add(time.Hour),
				Price: r.PvEstimate * 1e3,
			}
			if r.Period.Duration() != time.Hour {
				for i, r := range data {
					if r.Start.Equal(rr.Start) {
						data[i].Price = (r.Price + rr.Price) / 2
						continue NEXT
					}
				}
			}
			data = append(data, rr)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Solcast) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Solcast) Type() api.TariffType {
	return api.TariffTypeSolar
}
