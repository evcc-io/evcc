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
	mux     sync.Mutex
	log     *util.Logger
	homeID  string
	unit    string
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
		Token  string
		HomeID string
		Unit   string
		Cheap  any // TODO deprecated
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("tibber").Redact(cc.Token, cc.HomeID)

	t := &Tibber{
		log:    log,
		homeID: cc.HomeID,
		unit:   cc.Unit,
		client: tibber.NewClient(log, cc.Token),
	}

	if t.homeID == "" || t.unit == "" {
		home, err := t.client.DefaultHome(t.homeID)
		if err != nil {
			return nil, err
		}

		if t.homeID == "" {
			t.homeID = home.ID
		}
		if t.unit == "" {
			t.unit = home.CurrentSubscription.PriceInfo.Current.Currency
		}
	}

	// TODO deprecated
	if cc.Cheap != nil {
		t.log.WARN.Println("cheap rate configuration has been replaced by target charging and is deprecated")
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
		ar := api.Rate{
			Start: r.StartsAt.Local(),
			End:   r.StartsAt.Add(time.Hour).Local(),
			Price: r.Total,
		}
		data = append(data, ar)
	}
	return data
}

// Unit implements the api.Tariff interface
func (t *Tibber) Unit() string {
	return t.unit
}

// Rates implements the api.Tariff interface
func (t *Tibber) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}
