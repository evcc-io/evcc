package tariff

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/octopus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Octopus struct {
	mux     sync.Mutex
	log     *util.Logger
	uri     string
	region  string
	data    api.Rates
	updated time.Time
}

var _ api.Tariff = (*Octopus)(nil)

func init() {
	registry.Add("octopusenergy", NewOctopusFromConfig)
}

func NewOctopusFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Region string
		Tariff string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Region == "" {
		return nil, errors.New("missing region")
	}
	if cc.Tariff == "" {
		return nil, errors.New("missing tariff code")
	}

	t := &Octopus{
		log:    util.NewLogger("octopus"),
		uri:    octopus.ConstructRatesAPI(cc.Tariff, cc.Region),
		region: cc.Tariff,
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Octopus) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

	for ; true; <-time.Tick(time.Hour) {
		var res octopus.UnitRates

		if err := backoff.Retry(func() error {
			return client.GetJSON(t.uri, &res)
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		t.data = make(api.Rates, 0, len(res.Results))
		for _, r := range res.Results {
			ar := api.Rate{
				Start: r.ValidityStart,
				End:   r.ValidityEnd,
				// UnitRates are supplied inclusive of tax, though this could be flipped easily with a config flag.
				Price: r.PriceInclusiveTax / 1e2,
			}
			t.data = append(t.data, ar)
		}
		t.data.Sort()

		t.mux.Unlock()
	}
}

// Unit implements the api.Tariff interface
// Stubbed because supplier always works in GBP
func (t *Octopus) Unit() string {
	return "GBP"
}

// Rates implements the api.Tariff interface
func (t *Octopus) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}

// Type implements the api.Tariff interface
func (t *Octopus) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
