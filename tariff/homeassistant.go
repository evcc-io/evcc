package tariff

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant provides the solar forecast of the Home Assistant Energy dashboard
type HomeAssistant struct {
	log  *util.Logger
	conn *homeassistant.Connection
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*HomeAssistant)(nil)

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

func NewHomeAssistantFromConfig(other map[string]any) (api.Tariff, error) {
	cc := struct {
		homeassistant.Config `mapstructure:",squash"`
		Interval             time.Duration
	}{
		Interval: time.Hour,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("homeassistant")

	conn, err := cc.NewConnection(log)
	if err != nil {
		return nil, err
	}

	t := &HomeAssistant{
		log:  log,
		conn: conn,
		data: util.NewMonitor[api.Rates](2 * cc.Interval),
	}

	done := make(chan error)
	go t.run(cc.Interval, done)

	if err := <-done; err != nil {
		return nil, err
	}

	return t, nil
}

func (t *HomeAssistant) run(interval time.Duration, done chan error) {
	var once sync.Once

	for ; true; <-time.Tick(interval) {
		var data api.Rates

		if err := backoff.Retry(func() error {
			res, err := t.conn.SolarForecasts(context.Background())
			if err != nil {
				return err
			}
			if data, err = solarForecastRates(res); err != nil {
				return backoff.Permanent(err)
			}
			if len(data) == 0 {
				return backoff.Permanent(errors.New("no solar forecast configured in the energy dashboard"))
			}
			return nil
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}
			t.log.ERROR.Println(err)
			continue
		}

		mergeRatesAfter(t.data, data, beginningOfDay())
		once.Do(func() { close(done) })
	}
}

// solarForecastRates sums the hourly energy of all forecasts, which equals the average power in W
func solarForecastRates(res map[string]homeassistant.SolarForecast) (api.Rates, error) {
	energy := make(map[time.Time]float64)

	for _, f := range res {
		for k, wh := range f.WhHours {
			ts, err := time.Parse(time.RFC3339, k)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
			energy[ts.Local()] += wh
		}
	}

	data := make(api.Rates, 0, len(energy))
	for ts, wh := range energy {
		data = append(data, api.Rate{
			Start: ts,
			End:   ts.Add(time.Hour),
			Value: wh,
		})
	}

	data.Sort()

	return data, nil
}

// Rates implements the api.Tariff interface
func (t *HomeAssistant) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *HomeAssistant) Type() api.TariffType {
	return api.TariffTypeSolar
}
