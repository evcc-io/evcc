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
	log    *util.Logger
	uri    string
	region string
	data   *util.Monitor[api.Rates]
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
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
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

		data := make(api.Rates, 0, len(res.Results))
		for _, r := range res.Results {
			ar := api.Rate{
				Start: r.ValidityStart,
				End:   r.ValidityEnd,
				// UnitRates are supplied inclusive of tax, though this could be flipped easily with a config flag.
				Price: r.PriceInclusiveTax / 1e2,
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Octopus) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Octopus) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
