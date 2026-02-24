package tariff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Supported regions
var supportedRegions = []string{
	"BE", "NL", "DE-LU", "FR", "CH",
	"SE4", "SE3", "SE1", "DK1", "DK2",
	"FI", "NO1", "NO2", "NO3", "NO4", "NO5",
	"LV", "LT", "PL", "PT", "RO", "RS",
	"SI", "SK", "HU", "AT", "CZ", "HR", "EE",
}

// Stekker provider
type Stekker struct {
	*embed
	region   string
	interval time.Duration
	log      *util.Logger
	data     *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Stekker)(nil)

func init() {
	registry.Add("stekker", NewStekkerFromConfig)
}

const stekkerURI = "https://stekker.app/epex-forecast"

// NewStekkerFromConfig creates provider from config
func NewStekkerFromConfig(other map[string]any) (api.Tariff, error) {
	var cc struct {
		embed  `mapstructure:",squash"`
		Region string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if !slices.Contains(supportedRegions, cc.Region) {
		return nil, fmt.Errorf("unsupported region: %s", cc.Region)
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	interval := time.Hour

	switch cc.Region {
	case "BE":
		cc.Region = "BE-900"
		interval = 15 * time.Minute
	case "NL":
		cc.Region = "NL-900"
		interval = 15 * time.Minute
	}

	t := &Stekker{
		embed:    &cc.embed,
		region:   cc.Region,
		interval: interval,
		log:      util.NewLogger("stekker"),
		data:     util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return runOrError(t)
}

func (t *Stekker) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)

	for tick := time.Tick(time.Hour); ; <-tick {
		url := fmt.Sprintf("%s?advanced_view=&region=%s&unit=MWh", stekkerURI, t.region)
		resp, err := client.Get(url)
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println("http error:", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			once.Do(func() { done <- fmt.Errorf("http status %d", resp.StatusCode) })
			t.log.ERROR.Printf("http status %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			once.Do(func() { done <- err })
			t.log.ERROR.Println("parse error:", err)
			continue
		}
		resp.Body.Close()

		val, ok := doc.Find("[data-epex-forecast-graph-data-value]").Attr("data-epex-forecast-graph-data-value")
		if !ok {
			once.Do(func() { done <- fmt.Errorf("no forecast attribute found") })
			t.log.ERROR.Println("no forecast attribute found")
			continue
		}

		raw := strings.ReplaceAll(val, "&quot;", "\"")

		var data []map[string]any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println("unmarshal error:", err)
			continue
		}

		var res api.Rates
		for _, series := range data {
			name, _ := series["name"].(string)
			if !(strings.Contains(name, "Market") || strings.Contains(name, "Forecast")) {
				continue
			}

			xs, _ := series["x"].([]any)
			ys, _ := series["y"].([]any)

			for i := range xs {
				xt, ok1 := xs[i].(string)
				yt, ok2 := ys[i].(float64)
				if !ok1 || !ok2 {
					continue
				}

				start, err := time.Parse(time.RFC3339, xt)
				if err != nil {
					continue
				}

				res = append(res, api.Rate{
					Start: start,
					End:   start.Add(t.interval),
					Value: t.totalPrice(yt/1000.0, start), // €/MWh → €/kWh
				})
			}
		}

		mergeRates(t.data, res)
		once.Do(func() { close(done) })
	}
}

// Rates implements api.Tariff
func (t *Stekker) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements api.Tariff
func (t *Stekker) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
