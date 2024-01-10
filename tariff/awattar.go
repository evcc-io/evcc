package tariff

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/awattar"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Awattar struct {
	*embed
	log  *util.Logger
	uri  string
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Awattar)(nil)

func init() {
	registry.Add("awattar", NewAwattarFromConfig)
}

func NewAwattarFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Region string
	}{
		Region: "DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &Awattar{
		embed: &cc.embed,
		log:   util.NewLogger("awattar"),
		uri:   fmt.Sprintf(awattar.RegionURI, strings.ToLower(cc.Region)),
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Awattar) run(done chan error) {
	var once sync.Once
	bo := newBackoff()
	client := request.NewHelper(t.log)

	for ; true; <-time.Tick(time.Hour) {
		var res awattar.Prices

		if err := backoff.Retry(func() error {
			return client.GetJSON(t.uri, &res)
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Data))
		for _, r := range res.Data {
			ar := api.Rate{
				Start: r.StartTimestamp.Local(),
				End:   r.EndTimestamp.Local(),
				Price: t.totalPrice(r.Marketprice / 1e3),
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Awattar) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Awattar) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
