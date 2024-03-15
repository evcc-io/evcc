package tariff

import (
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/smartenergy"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type SmartEnergy struct {
	*embed
	log  *util.Logger
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*SmartEnergy)(nil)

func init() {
	registry.Add("smartenergy", NewSmartEnergyFromConfig)
}

func NewSmartEnergyFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed `mapstructure:",squash"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &SmartEnergy{
		embed: &cc.embed,
		log:   util.NewLogger("smartenergy"),
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *SmartEnergy) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

	for ; true; <-time.Tick(time.Hour) {
		var res smartenergy.Prices

		if err := backoff.Retry(func() error {
			return client.GetJSON(smartenergy.URI, &res)
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Data))
		for _, r := range res.Data {
			ar := api.Rate{
				Start: r.Date.Local(),
				End:   r.Date.Add(15 * time.Minute).Local(),
				Price: t.totalPrice(r.Value / 100),
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *SmartEnergy) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *SmartEnergy) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
