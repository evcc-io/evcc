package tariff

import (
	"context"
	"errors"
	"fmt"
	"maps"
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
			data = zeroFillBefore(data, beginningOfDay())
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

// solarForecastRates converts the energy per slot of all forecasts to average power in W.
// The slot length of a forecast is its smallest timestamp distance, defaulting to one hour.
func solarForecastRates(res map[string]homeassistant.SolarForecast) (api.Rates, error) {
	var slots api.Rates
	bounds := make(map[time.Time]struct{})

	for _, f := range res {
		energy := make(map[time.Time]float64, len(f.WhHours))
		for k, wh := range f.WhHours {
			ts, err := time.Parse(time.RFC3339, k)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
			// skip off-grid keys like Forecast.Solar sunrise/sunset that would shrink the slot length
			if !ts.Equal(ts.Truncate(15 * time.Minute)) {
				continue
			}
			energy[ts.Local()] += wh
		}

		ts := slices.SortedFunc(maps.Keys(energy), time.Time.Compare)

		d := time.Hour
		for i := 1; i < len(ts); i++ {
			if delta := ts[i].Sub(ts[i-1]); delta < d {
				d = delta
			}
		}

		for _, t := range ts {
			end := t.Add(d)
			slots = append(slots, api.Rate{Start: t, End: end, Value: energy[t] / d.Hours()})
			bounds[t], bounds[end] = struct{}{}, struct{}{}
		}
	}

	// sum slots of different granularity on the common time grid
	edges := slices.SortedFunc(maps.Keys(bounds), time.Time.Compare)

	// forecasts omit zero-energy periods, the optimizer expects gapless slots
	step := time.Hour
	for i := 0; i+1 < len(edges); i++ {
		step = min(step, edges[i+1].Sub(edges[i]))
	}

	data := make(api.Rates, 0, len(edges))
	for i := 0; i+1 < len(edges); i++ {
		start, end := edges[i], edges[i+1]

		var (
			sum   float64
			found bool
		)
		for _, s := range slots {
			if !s.Start.After(start) && !s.End.Before(end) {
				sum += s.Value
				found = true
			}
		}

		if found {
			data = append(data, api.Rate{Start: start, End: end, Value: sum})
			continue
		}

		for ; start.Before(end); start = start.Add(step) {
			slotEnd := start.Add(step)
			if slotEnd.After(end) {
				slotEnd = end
			}
			data = append(data, api.Rate{Start: start, End: slotEnd})
		}
	}

	return data, nil
}

// zeroFillBefore prepends zero-energy slots from the given time up to the first slot
func zeroFillBefore(data api.Rates, from time.Time) api.Rates {
	if len(data) == 0 {
		return data
	}

	step := data[0].End.Sub(data[0].Start)

	var pre api.Rates
	for ts := data[0].Start.Add(-step); !ts.Before(from); ts = ts.Add(-step) {
		pre = append(pre, api.Rate{Start: ts, End: ts.Add(step)})
	}
	slices.Reverse(pre)

	return append(pre, data...)
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
