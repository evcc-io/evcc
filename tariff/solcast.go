package tariff

import (
	"errors"
	"fmt"
	"net/http"
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
}

var _ api.Tariff = (*Solcast)(nil)

func init() {
	registry.Add("solcast", NewSolcastFromConfig)
}

func NewSolcastFromConfig(other map[string]any) (api.Tariff, error) {
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
	}

	t.Client.Transport = transport.BearerAuth(cc.Token, t.Client.Transport)

	done := make(chan error)
	go t.run(cc.Interval, done)

	if err := <-done; err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Solcast) run(interval time.Duration, done chan error) {
	var once sync.Once

	for ; true; <-time.Tick(interval) {
		// ensure we don't run when not needed, but execute once at startup
		select {
		case <-t.data.Done():
			if !t.fromTo.IsActive(time.Now().Hour()) {
				// keep cached forecast alive while fetching is paused
				mergeRatesAfter(t.data, nil, beginningOfDay())
				continue
			}
		default:
		}

		var res solcast.Forecasts

		if err := backoff.Retry(func() error {
			uri := fmt.Sprintf("https://api.solcast.com.au/rooftop_sites/%s/forecasts?period=PT30M&format=json&hours=96", t.site)
			err := t.GetJSON(uri, &res)
			// HTTP 429: Solcast does not document a per-second or concurrency limit,
			// only 10 requests/day for hobbyist accounts. However, when two rooftop
			// sites are configured, both goroutines bypass the fromTo window check on
			// their first iteration (via the default: branch) and fire simultaneously
			// at startup — potentially triggering a server-side burst/concurrency guard.
			// The Home Assistant Solcast integration observed the same behaviour and
			// works around it with randomised request delays:
			// https://github.com/BJReplay/ha-solcast-solar
			//
			// Log all available rate-limit headers to determine whether this is a
			// transient concurrency collision (small Retry-After) or daily quota
			// exhaustion (large Retry-After), then return a retryable error so the
			// existing exponential backoff (bo(), capped at 1 min) can recover.
			if se, ok := errors.AsType[*request.StatusError](err); ok && se.StatusCode() == http.StatusTooManyRequests {
				resp := se.Response()
				t.log.DEBUG.Printf("Solcast 429: Retry-After=%s X-RateLimit-Limit=%s X-RateLimit-Remaining=%s",
					resp.Header.Get("Retry-After"),
					resp.Header.Get("X-RateLimit-Limit"),
					resp.Header.Get("X-RateLimit-Remaining"),
				)
				return err // retryable: let bo() handle backoff
			}
			return backoffPermanentError(err)
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}
			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		data := make(api.Rates, 0, len(res.Forecasts))

	NEXT:
		for _, r := range res.Forecasts {
			start := now.With(r.PeriodEnd).BeginningOfHour().Local()
			rr := api.Rate{
				Start: start,
				End:   start.Add(time.Hour),
				Value: r.PvEstimate * 1e3,
			}
			if r.Period.Duration() != time.Hour {
				for i, r := range data {
					if r.Start.Equal(rr.Start) {
						data[i].Value = (r.Value + rr.Value) / 2
						continue NEXT
					}
				}
			}
			data = append(data, rr)
		}

		mergeRatesAfter(t.data, data, beginningOfDay())
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
