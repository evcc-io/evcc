package tariff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/corrently"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type GrünStromIndex struct {
	*request.Helper
	log  *util.Logger
	zip  string
	data *util.Monitor[api.Rates]
}

var _ api.Tariff = (*GrünStromIndex)(nil)

func init() {
	registry.Add("grünstromindex", NewGrünStromIndexFromConfig)
}

func NewGrünStromIndexFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Zip   string
		Token string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("gsi").Redact(cc.Zip, cc.Token)

	t := &GrünStromIndex{
		log:    log,
		zip:    cc.Zip,
		Helper: request.NewHelper(log),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: corrently.TokenSource(request.NewHelper(log), &oauth2.Token{AccessToken: cc.Token}),
	}

	startupErr := make(chan error)
	go t.run(startupErr)
	err := <-startupErr

	return t, err
}

func (t *GrünStromIndex) fetchForecast() (corrently.Forecast, error) {
	uri := fmt.Sprintf("https://api.corrently.io/v2.0/gsi/prediction?zip=%s", t.zip)

	var res corrently.Forecast
	err := t.GetJSON(uri, &res)

	return res, request.BackoffDefaultHttpStatusCodesPermanently(
		// do not stop the backoff handling when the API is down
		request.BackoffHttpStatusCode(http.StatusInternalServerError, false),
		request.BackoffHttpStatusCode(http.StatusTooManyRequests, false),
	)(err)
}

func (t *GrünStromIndex) run(startupErr chan<- error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		retries := 0
		res, err := backoff.Retry(context.Background(),
			t.fetchForecast,
			backoff.WithMaxElapsedTime(59*time.Minute),
			backoff.WithBackOff(&backoff.ExponentialBackOff{
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Minute,
			}),
			backoff.WithNotify(func(_ error, _ time.Duration) {
				retries++
				if retries >= 3 {
					// we are stuck retrying non-permanent errors -> no need to delay startup
					once.Do(func() { close(startupErr) })
				}
			},
			))

		if err == nil && res.Err {
			if s, ok := res.Message.(string); ok {
				err = errors.New(s)
			} else {
				err = api.ErrNotAvailable
			}
		}

		if err != nil {
			once.Do(func() { startupErr <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Forecast))
		for _, r := range res.Forecast {
			data = append(data, api.Rate{
				Start: time.UnixMilli(r.Timeframe.Start).Local(),
				End:   time.UnixMilli(r.Timeframe.End).Local(),
				Value: float64(r.Co2GStandard),
			})
		}

		mergeRates(t.data, data)
		once.Do(func() { close(startupErr) })
	}
}

// Rates implements the api.Tariff interface
func (t *GrünStromIndex) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *GrünStromIndex) Type() api.TariffType {
	return api.TariffTypeCo2
}
