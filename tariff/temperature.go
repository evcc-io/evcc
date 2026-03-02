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

// openMeteoResponse is the JSON response from the Open-Meteo API.
// We only decode the fields we need.
type openMeteoResponse struct {
	Hourly struct {
		Time          []string  `json:"time"`
		Temperature2m []float64 `json:"temperature_2m"`
	} `json:"hourly"`
}

// Temperature fetches hourly outdoor temperature forecasts from the Open-Meteo API
// and exposes them as api.Rates where Rate.Value is the temperature in °C.
type Temperature struct {
	log       *util.Logger
	latitude  float64
	longitude float64
	data      *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Temperature)(nil)

func init() {
	registry.Add("open-meteo", NewTemperatureFromConfig)
}

// NewTemperatureFromConfig creates a Temperature tariff from configuration.
//
// Example evcc.yaml:
//
//	tariffs:
//	  temperature:
//	    type: open-meteo
//	    latitude: 48.1
//	    longitude: 11.6
func NewTemperatureFromConfig(other map[string]any) (api.Tariff, error) {
	cc := struct {
		Latitude  float64 `mapstructure:"latitude"`
		Longitude float64 `mapstructure:"longitude"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Latitude < -90 || cc.Latitude > 90 || cc.Longitude < -180 || cc.Longitude > 180 {
		return nil, fmt.Errorf("open-meteo: latitude must be in [-90, 90] and longitude in [-180, 180]")
	}

	t := &Temperature{
		log:       util.NewLogger("open-meteo"),
		latitude:  cc.Latitude,
		longitude: cc.Longitude,
		data:      util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.log.INFO.Printf("configured for lat=%.4f lon=%.4f", cc.Latitude, cc.Longitude)

	return runOrError(t)
}

func (t *Temperature) run(done chan error) {
	var once sync.Once

	client := request.NewHelper(t.log)

	for ; true; <-time.Tick(time.Hour) {
		uri := fmt.Sprintf(
			"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m&past_days=7&forecast_days=3&timezone=UTC",
			t.latitude, t.longitude,
		)

		var res openMeteoResponse
		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Hourly.Time))
		for i, tsStr := range res.Hourly.Time {
			if i >= len(res.Hourly.Temperature2m) {
				break
			}

			// Open-Meteo returns ISO 8601 UTC strings like "2024-01-15T14:00" when timezone=UTC
			ts, err := time.ParseInLocation("2006-01-02T15:04", tsStr, time.UTC)
			if err != nil {
				t.log.WARN.Printf("cannot parse timestamp %q: %v", tsStr, err)
				continue
			}

			data = append(data, api.Rate{
				Start: ts,
				End:   ts.Add(time.Hour),
				Value: res.Hourly.Temperature2m[i],
			})
		}

		if len(data) > 0 {
			t.log.DEBUG.Printf("updated %d hourly slots (%s … %s)",
				len(data),
				data[0].Start.Local().Format("2006-01-02 15:04"),
				data[len(data)-1].Start.Local().Format("2006-01-02 15:04"),
			)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface.
// Each Rate.Value is the outdoor temperature in °C for that hour.
func (t *Temperature) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface.
func (t *Temperature) Type() api.TariffType {
	return api.TariffTypeTemperature
}
