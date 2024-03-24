package tariff

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/elering"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Elering struct {
	*embed
	log    *util.Logger
	region string
	data   *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Elering)(nil)

func init() {
	registry.Add("elering", NewEleringFromConfig)
}

func NewEleringFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed  `mapstructure:",squash"`
		Region string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Region == "" {
		return nil, errors.New("missing region")
	}

	t := &Elering{
		embed:  &cc.embed,
		log:    util.NewLogger("Elering"),
		region: strings.ToLower(cc.Region),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Elering) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var res elering.NpsPrice

		ts := time.Now().Truncate(time.Hour)
		uri := fmt.Sprintf("%s/nps/price?start=%s&end=%s", elering.URI,
			url.QueryEscape(ts.Format(time.RFC3339)),
			url.QueryEscape(ts.Add(48*time.Hour).Format(time.RFC3339)))

		if err := backoff.Retry(func() error {
			return client.GetJSON(uri, &res)
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Data[t.region]))
		for _, r := range res.Data[t.region] {
			ts := time.Unix(r.Timestamp, 0)

			ar := api.Rate{
				Start: ts.Local(),
				End:   ts.Add(time.Hour).Local(),
				Price: t.totalPrice(r.Price / 1e3),
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Elering) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Elering) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
