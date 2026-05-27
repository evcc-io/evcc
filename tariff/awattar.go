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

func NewAwattarFromConfig(other map[string]any) (api.Tariff, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Region string
	}{
		Region: "DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	t := &Awattar{
		embed: &cc.embed,
		log:   util.NewLogger("awattar"),
		uri:   fmt.Sprintf(awattar.RegionURI, strings.ToLower(cc.Region)),
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return runOrError(t)
}

func (t *Awattar) run(done chan error) {
	var once sync.Once

	client := request.NewHelper(t.log)

	for tick := time.Tick(time.Hour); ; <-tick {
		var res awattar.Prices

		// Awattar publishes prices for next day around 13:00 CET/CEST, so up to 35h of price data are available
		// To be on the safe side request a window of -2h and +48h, the API doesn't mind requesting more than available
		start := time.Now().Add(-2 * time.Hour).UnixMilli()
		end := time.Now().Add(48 * time.Hour).UnixMilli()
		uri := fmt.Sprintf("%s?start=%d&end=%d", t.uri, start, end)

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Data))
		for _, r := range res.Data {
			ar := api.Rate{
				Start: r.StartTimestamp.Local(),
				End:   r.EndTimestamp.Local(),
				Value: t.totalPrice(r.Marketprice/1e3, r.StartTimestamp),
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
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
