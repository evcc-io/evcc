package tariff

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/solcast"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jinzhu/now"
)

type Solcast struct {
	*request.Helper
	log    *util.Logger
	site   string
	fromTo FromTo
	data   *util.Monitor[api.Rates]
	cache  *SolarCacheManager
}

var _ api.Tariff = (*Solcast)(nil)

func init() {
	registry.Add("solcast", NewSolcastFromConfig)
}

func NewSolcastFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		Site     string
		Token    string
		Interval time.Duration
		FromTo   `mapstructure:",squash"`
	}{
		Interval: 3 * time.Hour,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Site == "" {
		return nil, errors.New("missing site id")
	}

	if cc.Token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("solcast").Redact(cc.Token)

	t := &Solcast{
		log:    log,
		site:   cc.Site,
		Helper: request.NewHelper(log),
		fromTo: cc.FromTo,
		data:   util.NewMonitor[api.Rates](2 * cc.Interval),
		cache:  NewSolarCacheManager("solcast", cc),
	}

	t.Client.Transport = transport.BearerAuth(cc.Token, t.Client.Transport)

	done := make(chan error)
	go t.run(cc.Interval, done)
	err := <-done

	return t, err
}

func (t *Solcast) run(interval time.Duration, done chan error) {
	var once sync.Once

	// Try to load from cache on startup
	loadSolarCacheWithDelay(t.cache, t.data, t.log, interval, done, &once)

	for ; true; <-time.Tick(interval) {
		// ensure we don't run when not needed, but execute once at startup
		select {
		case <-t.data.Done():
			if !t.fromTo.IsActive(time.Now().Hour()) {
				continue
			}
		default:
		}

		var res solcast.Forecasts

		if err := backoff.Retry(func() error {
			uri := fmt.Sprintf("https://api.solcast.com.au/rooftop_sites/%s/forecasts?period=PT30M&format=json", t.site)
			return backoffPermanentError(t.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		processedRates := make(api.Rates, 0, len(res.Forecasts))

	NEXT:
		for _, r := range res.Forecasts {
			start := now.With(r.PeriodEnd).BeginningOfHour().Local()
			rr := api.Rate{
				Start: start,
				End:   start.Add(time.Hour),
				Value: r.PvEstimate * 1e3,
			}
			if r.Period.Duration() != time.Hour {
				for i, r := range processedRates {
					if r.Start.Equal(rr.Start) {
						processedRates[i].Value = (r.Value + rr.Value) / 2
						continue NEXT
					}
				}
			}
			processedRates = append(processedRates, rr)
		}

		// Cache the processed data
		if err := t.cache.Set(processedRates); err != nil {
			t.log.DEBUG.Printf("failed to cache forecast data: %v", err)
		}

		// Update stored rates
		mergeRatesAfter(t.data, processedRates, beginningOfDay())
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Solcast) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Solcast) Type() api.TariffType {
	return api.TariffTypeSolar
}
