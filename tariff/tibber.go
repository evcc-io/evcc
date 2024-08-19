package tariff

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

type Tibber struct {
	*embed
	log    *util.Logger
	homeID string
	client *tibber.Client
	data   *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Tibber)(nil)

func init() {
	registry.Add("tibber", NewTibberFromConfig)
}

func NewTibberFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed  `mapstructure:",squash"`
		Token  string
		HomeID string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("tibber").Redact(cc.Token, cc.HomeID)

	t := &Tibber{
		embed:  &cc.embed,
		log:    log,
		homeID: cc.HomeID,
		client: tibber.NewClient(log, cc.Token),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	if t.homeID == "" {
		home, err := t.client.DefaultHome(t.homeID)
		if err != nil {
			return nil, err
		}

		t.homeID = home.ID
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Tibber) run(done chan error) {
	var once sync.Once

	v := map[string]interface{}{
		"id": graphql.ID(t.homeID),
	}

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var res struct {
			Viewer struct {
				Home struct {
					ID                  string
					TimeZone            string
					CurrentSubscription tibber.Subscription
				} `graphql:"home(id: $id)"`
			}
		}

		if err := backoff.Retry(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
			defer cancel()
			return t.client.Query(ctx, &res, v)
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		pi := res.Viewer.Home.CurrentSubscription.PriceInfo
		data := append(t.rates(pi.Today), t.rates(pi.Tomorrow)...)

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

func (t *Tibber) rates(pi []tibber.Price) api.Rates {
	data := make(api.Rates, 0, len(pi))
	for _, r := range pi {
		price := r.Total
		if t.Charges != 0 || t.Tax != 0 {
			price = t.totalPrice(r.Energy)
		}
		ar := api.Rate{
			Start: r.StartsAt.Local(),
			End:   r.StartsAt.Add(time.Hour).Local(),
			Price: price,
		}
		data = append(data, ar)
	}
	return data
}

// Rates implements the api.Tariff interface
func (t *Tibber) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Tibber) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
