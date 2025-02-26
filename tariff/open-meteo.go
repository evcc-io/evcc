package tariff

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	openmeteo "github.com/evcc-io/evcc/tariff/open-meteo"
	"github.com/evcc-io/evcc/util"
)

///////////////////////////////////////////////////////////////////////////////////////////
///// DEBUG VERISON
///////////////////////////////////////////////////////////////////////////////////////////

type OpenMeteo struct {
	*openmeteo.OpenMeteo
	site string
	log  *util.Logger
	Data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*OpenMeteo)(nil)

func init() {
	registry.Add("open-meteo", NewOpenMeteoFromOpenMeteo)
}

func NewOpenMeteoFromOpenMeteo(other map[string]interface{}) (api.Tariff, error) {
	cc := openmeteo.OpenMeteo{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if len(cc.Latitude) == 0 || len(cc.Longitude) == 0 || len(cc.Azimuth) == 0 || len(cc.Declination) == 0 || len(cc.DcKwp) == 0 || cc.AcKwp == 0 {
		return nil, errors.New("missing required parameters")
	}

	log := util.NewLogger("open-meteo")

	t := &OpenMeteo{
		OpenMeteo: openmeteo.NewOpenMeteo(log, cc),
		site:      "open-meteo",
		log:       log,
		Data:      util.NewMonitor[api.Rates](2 * time.Minute),
	}

	done := make(chan error)
	go t.run(cc.Interval, done)
	err := <-done

	return t, err
}

func (t *OpenMeteo) run(interval time.Duration, done chan error) {
	var once sync.Once

	// Initial fetch
	if err := t.UpdateRates(); err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}
	once.Do(func() { close(done) })

	// Ticker to update rates at the specified interval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := t.UpdateRates(); err != nil {
				t.log.ERROR.Println(err)
			}
		}
	}
}

func (t *OpenMeteo) UpdateRates() error {
	if len(t.Latitude) == 0 {
		return fmt.Errorf("Latitude slice is empty")
	}

	for i := range t.Latitude {
		params := url.Values{
			"latitude":      {fmt.Sprintf("%f", t.Latitude[i])},
			"longitude":     {fmt.Sprintf("%f", t.Longitude[i])},
			"azimuth":       {fmt.Sprintf("%f", t.Azimuth[i])},
			"tilt":          {fmt.Sprintf("%f", t.Declination[i])},
			"hourly":        {"temperature_2m,global_tilted_irradiance,global_tilted_irradiance_instant"},
			"daily":         {"sunrise,sunset"},
			"forecast_days": {fmt.Sprintf("%d", t.ForecastDays)},
			"past_days":     {fmt.Sprintf("%d", t.PastDays)},
			"timezone":      {"auto"},
		}

		if t.ApiKey != "" {
			params.Set("apikey", t.ApiKey)
		}

		if t.WeatherModel != "" {
			params.Set("models", t.WeatherModel)
		}

		uri := fmt.Sprintf("%s/v1/forecast?%s", t.BaseURL, params.Encode())
		t.log.INFO.Printf("Requesting URL: %s", uri) // Log the request URL
		var res openmeteo.Response
		t.log.INFO.Printf("ZWEI %s", res) // Log HTTP request error
		if err := backoff.Retry(func() error {
			resp, err := t.Get(uri)
			if err != nil {
				t.log.ERROR.Printf("HTTP request error: %v", err) // Log HTTP request error
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.log.ERROR.Printf("Unexpected status: %s", resp.Status) // Log unexpected status
				return fmt.Errorf("unexpected status: %s", resp.Status)
			}

			return json.NewDecoder(resp.Body).Decode(&res)
		}, backoff.NewExponentialBackOff()); err != nil {
			t.log.ERROR.Printf("Backoff retry error: %v", err) // Log backoff retry error
			return err
		}

		// Log the response data for debugging
		responseData, err := json.Marshal(res)
		if err != nil {
			t.log.ERROR.Printf("JSON marshal error: %v", err) // Log JSON marshal error
			return err
		}
		t.log.INFO.Printf("Response data: %s", responseData)

		// Pass the response to the calculate function in calc.go and store the result in the OpenMeteo struct
		if err := t.Calculate(res); err != nil {
			t.log.ERROR.Printf("Calculate error: %v", err)
			return err
		} else {
			t.log.INFO.Printf("Calculate success")
		}
	}
	return nil
}

// Rates implements the api.Tariff interface
func (t *OpenMeteo) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.Data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *OpenMeteo) Type() api.TariffType {
	return api.TariffTypeSolar
}
