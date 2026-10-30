package tariff

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://localvolts.com/wp-content/uploads/2026/10/LvAPI_20261002.pdf
const localVoltsURI = "https://api.localvolts.com/v2/customer/interval?NMI=%s&from=current&to=1day"

type localVoltsInterval struct {
	Circuit          string    `json:"circuit"`
	IntervalEnd      time.Time `json:"intervalEnd"`
	IntervalDuration int       `json:"intervalDuration,string"` // minutes, sent as string despite the docs
	FlexUp           float64   `json:"flexUp"`                  // c/kWh
}

type LocalVolts struct {
	*embed
	*request.Helper
	log     *util.Logger
	uri     string
	channel string
	data    *util.Monitor[api.Rates]
}

var _ api.Tariff = (*LocalVolts)(nil)

func init() {
	registry.Add("localvolts", NewLocalVoltsFromConfig)
}

func NewLocalVoltsFromConfig(other map[string]any) (api.Tariff, error) {
	var cc struct {
		embed   `mapstructure:",squash"`
		ApiKey  string
		Partner string
		Nmi     string
		Channel string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ApiKey == "" || cc.Partner == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Nmi == "" {
		return nil, errors.New("missing nmi")
	}

	if cc.Channel == "" {
		return nil, errors.New("missing channel")
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	log := util.NewLogger("localvolts").Redact(cc.ApiKey)

	t := &LocalVolts{
		embed:   &cc.embed,
		log:     log,
		Helper:  request.NewHelper(log),
		uri:     fmt.Sprintf(localVoltsURI, cc.Nmi),
		channel: cc.Channel,
		data:    util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &transport.Decorator{
		Base: t.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "apikey " + cc.ApiKey,
			"partner":       cc.Partner,
		}),
	}

	return runOrError(t)
}

func (t *LocalVolts) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Minute); ; <-tick {
		var res []localVoltsInterval

		if err := backoff.Retry(func() error {
			return backoffPermanentError(t.GetJSON(t.uri, &res))
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}

			t.log.ERROR.Println(err)
			continue
		}

		data := t.rates(res)
		if len(data) == 0 {
			err := fmt.Errorf("no data for channel %s", t.channel)
			if reportError(&once, done, err) {
				return
			}

			t.log.ERROR.Println(err)
			continue
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// rates averages the 5-minute NEM intervals of the configured circuit into 15-minute slots.
// flexUp is the marginal price per kWh: positive means cost on Import and revenue on Export,
// which matches evcc's sign convention for grid and feed-in tariffs.
func (t *LocalVolts) rates(res []localVoltsInterval) api.Rates {
	type bucket struct {
		sum float64
		n   int
	}
	buckets := make(map[time.Time]*bucket)

	for _, r := range res {
		if !strings.EqualFold(r.Circuit, t.channel) {
			continue
		}

		start := r.IntervalEnd.Add(-time.Duration(r.IntervalDuration) * time.Minute).Local()
		slot := start.Truncate(SlotDuration)

		b, ok := buckets[slot]
		if !ok {
			b = &bucket{}
			buckets[slot] = b
		}

		b.sum += t.totalPrice(r.FlexUp/1e2, start)
		b.n++
	}

	data := make(api.Rates, 0, len(buckets))
	for slot, b := range buckets {
		data = append(data, api.Rate{
			Start: slot,
			End:   slot.Add(SlotDuration),
			Value: b.sum / float64(b.n),
		})
	}
	data.Sort()

	return data
}

// Rates implements the api.Tariff interface
func (t *LocalVolts) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *LocalVolts) Unit() string {
	return "AUD"
}

// Type implements the api.Tariff interface
func (t *LocalVolts) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
