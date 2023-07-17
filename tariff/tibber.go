package tariff

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/shurcooL/graphql"
	"golang.org/x/exp/slices"
)

type Tibber struct {
	*embed
	mux     sync.Mutex
	log     *util.Logger
	homeID  string
	client  *tibber.Client
	data    api.Rates
	updated time.Time
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
		Unit   string
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

	var res struct {
		Viewer struct {
			Home struct {
				ID                  string
				TimeZone            string
				CurrentSubscription tibber.Subscription
			} `graphql:"home(id: $id)"`
		}
	}

	v := map[string]interface{}{
		"id": graphql.ID(t.homeID),
	}

	for ; true; <-time.Tick(time.Hour) {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		err := t.client.Query(ctx, &res, v)
		cancel()

		if err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		pi := res.Viewer.Home.CurrentSubscription.PriceInfo
		t.data = make(api.Rates, 0, len(pi.Today)+len(pi.Tomorrow))
		t.data = append(t.rates(pi.Today), t.rates(pi.Tomorrow)...)

		t.mux.Unlock()
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
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}

// Type returns the tariff type
func (t *Tibber) Type() api.TariffType {
	return api.TariffTypePriceDynamic
}
