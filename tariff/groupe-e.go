package tariff

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type GroupeE struct {
	*embed
	log  *util.Logger
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*GroupeE)(nil)

func init() {
	registry.Add("groupe-e", NewGroupeEFromConfig)
}

func NewGroupeEFromConfig(other map[string]interface{}) (api.Tariff, error) {
	t := &GroupeE{
		log:  util.NewLogger("groupe-e"),
		data: util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *GroupeE) run(done chan error) {
	var once sync.Once

	client := request.NewHelper(t.log)

	for tick := time.Tick(time.Hour); ; <-tick {
		var res []struct {
			StartTimestamp time.Time `json:"start_timestamp"`
			EndTimestamp   time.Time `json:"end_timestamp"`
			VarioPlus      float64   `json:"vario_plus"`
		}

		start := time.Now().Truncate(time.Hour)
		uri := fmt.Sprintf("https://api.tariffs.groupe-e.ch/v1/tariffs?start_timestamp=%s&end_timestamp=%s", start.Format(time.RFC3339), start.Add(48*time.Hour).Format(time.RFC3339))

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res))
		for _, r := range res {
			ar := api.Rate{
				Start: r.StartTimestamp.Local(),
				End:   r.EndTimestamp.Local(),
				Value: r.VarioPlus / 1e2, // Rp/kWh
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *GroupeE) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *GroupeE) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
