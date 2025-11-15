// Package tariff implements EKZ (Elektrizitätswerke des Kantons Zürich) tariff provider.
//
// This implementation fetches electricity pricing data from EKZ's API incl. fallback handling:
// - Fetches the configured tariff as primary data source
// - Always fetches integrated_400ST as fallback in case primary tariff is unavailable (as documented by EKZ)
// - Updates both tariffs hourly since dynamic tariffs are published day-ahead and don't change once announced
// - Automatically falls back to static rates during API outages or when dynamic rates aren't available
// - Supports all EKZ tariff types that return "integrated" (EKZ term for all price components added up) prices

package tariff

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/ekz"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("ekz", NewEkzFromConfig)
}

type Ekz struct {
	*embed
	log           *util.Logger
	mux           sync.Mutex
	uri           string
	tariffName    string
	fallbackRates api.Rates // standard tariff rates as fallback
	data          *util.Monitor[api.Rates]
}

func NewEkzFromConfig(other map[string]interface{}) (api.Tariff, error) {
	c := struct {
		embed      `mapstructure:",squash"`
		URI        string `mapstructure:"uri"`
		TariffName string `mapstructure:"tariff"`
	}{}
	log := util.NewLogger("ekz")

	if err := util.DecodeOther(other, &c); err != nil {
		return nil, err
	}

	// Set defaults
	if c.URI == "" {
		c.URI = ekz.URI
	}
	if c.TariffName == "" {
		c.TariffName = "integrated_400D"
	}

	// Warn if tariff doesn't start with "integrated_" as user likely wants total cost
	if !strings.HasPrefix(c.TariffName, "integrated_") {
		log.WARN.Printf("tariff '%s' does not start with 'integrated_' - this may not include all cost components (grid, regional fees, metering)", c.TariffName)
	}

	t := &Ekz{
		embed:      &c.embed,
		log:        log,
		uri:        c.URI,
		tariffName: c.TariffName,
		data:       util.NewMonitor[api.Rates](2 * time.Hour),
	}
	log.DEBUG.Println("creating EKZ tariff instance with URI", c.URI, "and tariff", c.TariffName)

	// Initialize embed configuration
	if err := t.embed.init(); err != nil {
		return nil, err
	}
	return runOrError(t)
}

func (t *Ekz) buildURL(name string) string {
	now := time.Now()
	start := now.Truncate(15 * time.Minute)
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 23, 59, 59, 0, now.Location())
	qs := fmt.Sprintf(
		"?tariff_name=%s&start_timestamp=%s&end_timestamp=%s",
		name,
		url.QueryEscape(start.Format(time.RFC3339)),
		url.QueryEscape(end.Format(time.RFC3339)),
	)
	return t.uri + qs
}

// parseRates returns only CHF/kWh entries and applies totalPrice.
func (t *Ekz) parseRates(res ekz.TariffResponse) api.Rates {
	var out api.Rates
	for _, e := range res.Prices {
		for _, ir := range e.Integrated {
			if ir.Unit == "CHF/kWh" {
				out = append(out, api.Rate{
					Start: e.StartTimestamp,
					End:   e.EndTimestamp,
					Value: t.totalPrice(ir.Value, e.StartTimestamp),
				})
				break
			}
		}
	}
	return out
}

func (t *Ekz) run(done chan error) {
	t.log.DEBUG.Println("start")
	client := request.NewHelper(t.log)
	tick := time.NewTicker(time.Hour)
	defer tick.Stop()

	// trigger initial fetch and close done on first success/fail
	if err := t.updateAll(client); err != nil {
		done <- err
		return
	}
	close(done)

	for range tick.C {
		t.updateAll(client)
	}
}

// updateAll fetches fallback then main rates.
func (t *Ekz) updateAll(client *request.Helper) error {
	t.fetchFallback(client)
	return t.fetchMain(client)
}

func (t *Ekz) fetchMain(client *request.Helper) error {
	url := t.buildURL(t.tariffName)
	var res ekz.TariffResponse
	if err := client.GetJSON(url, &res); err != nil {
		t.log.ERROR.Printf("failed main fetch: %v", err)
		if !t.useFallbackRates() {
			return err
		}
		return nil
	}
	rates := t.parseRates(res)
	if len(rates) == 0 {
		t.log.WARN.Println("no CHF/kWh in main response")
		t.useFallbackRates()
		return nil
	}

	t.mux.Lock()
	t.data.Set(rates)
	t.mux.Unlock()
	t.log.DEBUG.Printf("main rates updated (%d)", len(rates))
	return nil
}

func (t *Ekz) fetchFallback(client *request.Helper) {
	url := t.buildURL("integrated_400ST")
	var res ekz.TariffResponse
	if err := client.GetJSON(url, &res); err != nil {
		t.log.ERROR.Printf("failed fallback fetch: %v", err)
		return
	}
	rates := t.parseRates(res)
	if len(rates) == 0 {
		t.log.WARN.Println("no CHF/kWh in fallback")
		return
	}

	t.mux.Lock()
	t.fallbackRates = rates
	t.mux.Unlock()
	t.log.DEBUG.Printf("fallback rates updated (%d)", len(rates))
}

// useFallbackRates applies fallback rates when main tariff is unavailable
func (t *Ekz) useFallbackRates() bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	if len(t.fallbackRates) > 0 {
		t.data.Set(t.fallbackRates)
		return true
	}
	t.log.WARN.Println("no fallback rates available")
	return false
}

func (t *Ekz) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Ekz) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
