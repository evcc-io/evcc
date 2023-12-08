package tariff

import (
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type EdfTempo struct {
	*embed
	log  *util.Logger
	uri  string
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*EdfTempo)(nil)

func init() {
	registry.Add("edf-tempo", NewEdfTempoFromConfig)
}

func NewEdfTempoFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed            `mapstructure:",squash"`
		Red, Blue, White float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &EdfTempo{
		embed: &cc.embed,
		log:   util.NewLogger("edf-tempo"),
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *EdfTempo) run(done chan error) {
	var once sync.Once
	bo := newBackoff()
	client := request.NewHelper(t.log)

	for ; true; <-time.Tick(time.Hour) {
		var res any

		uri := "https://digital.iservices.rte-france.com/open_api/tempo_like_supply_contract/v1/sandbox/tempo_like_calendars"
		// ts := time.Now().Truncate(time.Hour)
		// uri := fmt.Sprintf("https://digital.iservices.rte-france.com/open_api/tempo_like_supply_contract/v1/tempo_like_calendars?start_date=%s&end_date=%s",
		// 	url.QueryEscape(ts.Format(time.RFC3339)),
		// 	url.QueryEscape(ts.Add(48*time.Hour).Format(time.RFC3339)))

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &res))
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		// data := make(api.Rates, 0, len(res.Data))
		data := make(api.Rates, 0)
		// for _, r := range res.Data {
		// 	ar := api.Rate{
		// 		Start: r.StartTimestamp.Local(),
		// 		End:   r.EndTimestamp.Local(),
		// 		Price: t.totalPrice(r.Marketprice / 1e3),
		// 	}
		// 	data = append(data, ar)
		// }
		data.Sort()

		t.data.Set(data)
	}
}

// Rates implements the api.Tariff interface
func (t *EdfTempo) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *EdfTempo) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
