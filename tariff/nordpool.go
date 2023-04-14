package tariff

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/nordpool"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/exp/slices"
)

type Nordpool struct {
	mux     sync.Mutex
	log     *util.Logger
	unit    string
	region  string
	data    api.Rates
	updated time.Time
}

var _ api.Tariff = (*Nordpool)(nil)

func init() {
	registry.Add("nordpool", NewNordpoolFromConfig)
}

func NewNordpoolFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		Currency string
		Region   string
	}{
		Currency: "EUR",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Region == "" {
		return nil, errors.New("missing region")
	}

	t := &Nordpool{
		log:    util.NewLogger("nordpool"),
		unit:   cc.Currency,
		region: strings.ToLower(cc.Region),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Nordpool) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)

	for ; true; <-time.Tick(time.Hour) {
		var res nordpool.NpsPrice

		ts := time.Now().Truncate(time.Hour)
		uri := fmt.Sprintf("%s/nps/price?start=%s&end=%s", nordpool.URI,
			url.QueryEscape(ts.Format(time.RFC3339)),
			url.QueryEscape(ts.Add(2*48*time.Hour).Format(time.RFC3339)))

		if err := client.GetJSON(uri, &res); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		data := res.Data[t.region]

		t.data = make(api.Rates, 0, len(data))
		for _, r := range data {
			ts := time.Unix(r.Timestamp, 0)

			ar := api.Rate{
				Start: ts.Local(),
				End:   ts.Add(time.Hour).Local(),
				Price: r.Price / 100,
			}
			t.data = append(t.data, ar)
		}

		t.mux.Unlock()
	}
}

// Unit implements the api.Tariff interface
func (t *Nordpool) Unit() string {
	return t.unit
}

// Rates implements the api.Tariff interface
func (t *Nordpool) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}

// IsDynamic implements the api.Tariff interface
func (t *Nordpool) IsDynamic() bool {
	return true
}
