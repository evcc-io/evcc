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
)

type Tibber struct {
	mux    sync.Mutex
	log    *util.Logger
	homeID string
	cheap  float64
	client *tibber.Client
	data   []tibber.PriceInfo
}

var _ api.Tariff = (*Tibber)(nil)

func NewTibber(other map[string]interface{}) (*Tibber, error) {
	var cc struct {
		Token  string
		HomeID string
		Cheap  float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("tibber").Redact(cc.Token, cc.HomeID)

	t := &Tibber{
		log:    log,
		homeID: cc.HomeID,
		cheap:  cc.Cheap,
		client: tibber.NewClient(log, cc.Token),
	}

	if t.homeID == "" {
		var err error
		if t.homeID, err = t.client.DefaultHomeID(); err != nil {
			return nil, err
		}
	}

	go t.Run()

	return t, nil
}

func (t *Tibber) Run() {
	for ; true; <-time.NewTicker(time.Hour).C {
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

		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		err := t.client.Query(ctx, &res, v)
		cancel()

		if err != nil {
			t.log.ERROR.Println(err)
			continue
		}

		t.mux.Lock()
		t.data = res.Viewer.Home.CurrentSubscription.PriceInfo.Today
		t.mux.Unlock()
	}
}

func (t *Tibber) CurrentPrice() (float64, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	for i := len(t.data) - 1; i >= 0; i-- {
		pi := t.data[i]

		if pi.StartsAt.Before(time.Now()) {
			return pi.Total, nil
		}
	}
	return 0, errors.New("unable to find current tibber price")
}

func (t *Tibber) IsCheap() (bool, error) {
	price, err := t.CurrentPrice()
	return price <= t.cheap, err
}
