package tariff

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/greenely"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Greenely struct {
	*embed
	log    *util.Logger
	client *greenely.Client
	data   *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Greenely)(nil)

func init() {
	registry.Add("greenely", NewGreenelyFromConfig)
}

func NewGreenelyFromConfig(other map[string]any) (api.Tariff, error) {
	var cc struct {
		embed      `mapstructure:",squash"`
		Email      string
		Password   string
		FacilityID int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	log := util.NewLogger("greenely").Redact(cc.Email, cc.Password)

	t := &Greenely{
		embed:  &cc.embed,
		log:    log,
		client: greenely.NewClient(cc.Email, cc.Password),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	if cc.FacilityID != 0 {
		t.client.FacilityID = cc.FacilityID
	}

	return runOrError(t)
}

func (t *Greenely) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var resp greenely.SpotPrice

		if err := backoff.Retry(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
			defer cancel()

			if err := t.client.CheckAuth(ctx); err != nil {
				return err
			}

			r, err := t.client.GetSpotPrice(ctx, time.Now(), time.Now().Add(24*time.Hour))
			resp = r
			return err
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		mergeRates(t.data, t.rates(resp))
		once.Do(func() { close(done) })
	}
}

func fromMilliOreToKrona(val int) float64 {
	return float64(val) / 1000 / 100
}

func (t *Greenely) rates(sp greenely.SpotPrice) api.Rates {
	data := make(api.Rates, 0, len(sp.Data)*4)
	for _, hourData := range sp.Data {
		start := hourData.LocalTime.Time.Local()

		prices := []int{
			hourData.QuartersPrices.Quarter1,
			hourData.QuartersPrices.Quarter2,
			hourData.QuartersPrices.Quarter3,
			hourData.QuartersPrices.Quarter4,
		}

		for _, p := range prices {
			end := start.Add(SlotDuration)

			data = append(data, api.Rate{
				Start: start,
				End:   end,
				Value: t.totalPrice(fromMilliOreToKrona(p), start),
			})

			start = end
		}
	}
	return data
}

// Rates implements the api.Tariff interface
func (t *Greenely) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Greenely) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
