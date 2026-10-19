package tariff

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const frankEnergieURI = "https://graphql.frankenergie.nl/"

const frankEnergieQuery = `query MarketPrices($date: String!, $resolution: PriceResolution!) {
  marketPrices(date: $date, resolution: $resolution) {
    electricityPrices { from till marketPrice allInPrice }
  }
}`

type FrankEnergie struct {
	*embed
	log    *util.Logger
	export bool
	loc    *time.Location
	data   *util.Monitor[api.Rates]
}

type frankEnergiePrice struct {
	From        time.Time
	Till        time.Time
	MarketPrice float64
	AllInPrice  float64
}

type frankEnergieResponse struct {
	Data struct {
		MarketPrices struct {
			ElectricityPrices []frankEnergiePrice
		}
	}
	Errors []struct {
		Message string
	}
}

var _ api.Tariff = (*FrankEnergie)(nil)

func init() {
	registry.Add("frank-energie", NewFrankEnergieFromConfig)
}

func NewFrankEnergieFromConfig(other map[string]any) (api.Tariff, error) {
	var cc struct {
		embed           `mapstructure:",squash"`
		TariffDirection string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	var export bool
	switch strings.ToLower(cc.TariffDirection) {
	case "", "import":
	case "export":
		export = true
	default:
		return nil, fmt.Errorf("invalid tariff direction: %s", cc.TariffDirection)
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		return nil, err
	}

	t := &FrankEnergie{
		embed:  &cc.embed,
		log:    util.NewLogger("frank-energie"),
		export: export,
		loc:    loc,
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return runOrError(t)
}

func (t *FrankEnergie) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)

	for tick := time.Tick(time.Hour); ; <-tick {
		today := time.Now().In(t.loc)

		var prices []frankEnergiePrice
		if err := backoff.Retry(func() error {
			var err error
			prices, err = t.prices(client, today)
			return err
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}

			t.log.ERROR.Println(err)
			continue
		}

		res := t.rates(prices)

		// day-ahead prices for tomorrow are published around 13:00
		next, err := t.prices(client, today.AddDate(0, 0, 1))
		if err == nil {
			res = append(res, t.rates(next)...)
		} else {
			t.log.TRACE.Println("tomorrow:", err)
		}

		t.data.SetFunc(func(old api.Rates) api.Rates {
			// keep previously fetched slots for tomorrow if the update failed
			if err != nil {
				res = append(res, ratesAfter(old, res[len(res)-1].End)...)
			}
			return mergeAfter(old, res, time.Now().Truncate(SlotDuration))
		})
		once.Do(func() { close(done) })
	}
}

func (t *FrankEnergie) prices(client *request.Helper, day time.Time) ([]frankEnergiePrice, error) {
	data := map[string]any{
		"query": frankEnergieQuery,
		"variables": map[string]string{
			"date":       day.Format(time.DateOnly),
			"resolution": "PT15M",
		},
	}

	req, err := request.New(http.MethodPost, frankEnergieURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var res frankEnergieResponse
	if err := client.DoJSON(req, &res); err != nil {
		return nil, err
	}

	if len(res.Errors) > 0 {
		return nil, errors.New(res.Errors[0].Message)
	}

	if len(res.Data.MarketPrices.ElectricityPrices) == 0 {
		return nil, errors.New("no prices")
	}

	return res.Data.MarketPrices.ElectricityPrices, nil
}

// ratesAfter returns the rates starting at or after ts
func ratesAfter(rates api.Rates, ts time.Time) api.Rates {
	var res api.Rates
	for _, r := range rates {
		if !r.Start.Before(ts) {
			res = append(res, r)
		}
	}
	return res
}

func (t *FrankEnergie) rates(prices []frankEnergiePrice) api.Rates {
	res := make(api.Rates, 0, len(prices))
	for _, p := range prices {
		price := p.AllInPrice
		if t.export || t.Charges != 0 || len(t.chargesZones) > 0 || t.Tax != 0 || t.Formula != "" {
			price = t.totalPrice(p.MarketPrice, p.From)
		}

		res = append(res, api.Rate{
			Start: p.From.Local(),
			End:   p.Till.Local(),
			Value: price,
		})
	}
	return res
}

// Rates implements the api.Tariff interface
func (t *FrankEnergie) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *FrankEnergie) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
