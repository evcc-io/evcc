package tariff

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/energinet"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Energinet struct {
	*embed
	mux     sync.Mutex
	log     *util.Logger
	region  string
	data    api.Rates
	updated time.Time
}

var _ api.Tariff = (*Energinet)(nil)

func init() {
	registry.Add("energinet", NewEnerginetFromConfig)
}

func NewEnerginetFromConfig(other map[string]interface{}) (api.Tariff, error) {
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

	t := &Energinet{
		embed:  &cc.embed,
		log:    util.NewLogger("energinet"),
		region: strings.ToLower(cc.Region),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Energinet) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

	for ; true; <-time.Tick(time.Hour) {
		var res energinet.Prices

		ts := time.Now().Truncate(time.Hour)
		uri := fmt.Sprintf(energinet.URI,
			ts.Format(time.RFC3339),
			ts.Add(24*time.Hour).Format(time.RFC3339),
			t.region)

		if err := backoff.Retry(func() error {
			return client.GetJSON(uri, &res)
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		t.data = make(api.Rates, 0, len(res.Records))
		for _, r := range res.Records {
			date, _ := time.Parse("2006-01-02T15:04:05", r.HourUTC)
			ar := api.Rate{
				Start: date.Local(),
				End:   date.Add(time.Hour).Local(),
				Price: t.totalPrice(r.SpotPriceDKK / 1e3),
			}
			t.data = append(t.data, ar)
		}

		t.mux.Unlock()
	}
}

// Rates implements the api.Tariff interface
func (t *Energinet) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}

// Type implements the api.Tariff interface
func (t *Energinet) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
