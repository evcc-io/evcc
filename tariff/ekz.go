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
	fallbackRates api.Rates // electricity_standard rates as fallback
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

func (t *Ekz) run(done chan error) {
	t.log.DEBUG.Println("start")
	var once sync.Once
	client := request.NewHelper(t.log)

	// Build URLs for main tariff and fallback using base URI + tariff parameter
	// Calculate time range: start from previous 15min slot, end at end of next day
	now := time.Now()
	startTime := now.Truncate(15 * time.Minute)
	endTime := time.Date(now.Year(), now.Month(), now.Day()+1, 23, 59, 59, 0, now.Location())
	
	startTimestamp := url.QueryEscape(startTime.Format("2006-01-02T15:04:05-07:00"))
	endTimestamp := url.QueryEscape(endTime.Format("2006-01-02T15:04:05-07:00"))
	
	mainURL := fmt.Sprintf("%s?tariff_name=%s&start_timestamp=%s&end_timestamp=%s", 
		t.uri, t.tariffName, startTimestamp, endTimestamp)
	fallbackURL := fmt.Sprintf("%s?tariff_name=integrated_400ST&start_timestamp=%s&end_timestamp=%s", 
		t.uri, startTimestamp, endTimestamp)

	for tick := time.Tick(time.Hour); ; <-tick {
		// First, try to fetch and update fallback rates (integrated_400ST)
		t.fetchFallbackRates(client, fallbackURL)

		// Then fetch main tariff rates
		var res ekz.TariffResponse
		t.log.DEBUG.Printf("fetching %s tariff data from %s", t.tariffName, mainURL)
		if err := client.GetJSON(mainURL, &res); err != nil {
			t.log.ERROR.Printf("failed to get %s tariff data from %s: %v", t.tariffName, mainURL, err)

			// Use fallback rates if available
			if t.useFallbackRates() {
				once.Do(func() { close(done) })
			} else {
				once.Do(func() { done <- err })
			}
			continue
		}

		t.log.DEBUG.Printf("received %d price entries from %s tariff", len(res.Prices), t.tariffName)

		t.mux.Lock()
		rates := make(api.Rates, 0, len(res.Prices))
		for _, entry := range res.Prices {
			// Find the CHF/kWh price from integrated rates (total cost including all components)
			t.log.DEBUG.Printf("processing entry with %d integrated rates", len(entry.Integrated))
			for i, rate := range entry.Integrated {
				t.log.DEBUG.Printf("  integrated rate[%d]: unit=%s, value=%f", i, rate.Unit, rate.Value)
				if rate.Unit == "CHF/kWh" {
					rates = append(rates, api.Rate{
						Start: entry.StartTimestamp,
						End:   entry.EndTimestamp,
						Value: t.totalPrice(rate.Value, entry.StartTimestamp),
					})
					break
				}
				// Skip CHF/M (monthly fixed costs) and other units
			}
		}

		if len(rates) > 0 {
			t.data.Set(rates)
			t.log.DEBUG.Printf("updated %s tariff with %d rates", t.tariffName, len(rates))
			once.Do(func() { close(done) })
		} else {
			t.log.WARN.Printf("no CHF/kWh rates found in %s tariff response", t.tariffName)
			if strings.HasPrefix(t.tariffName, "integrated_") {
				// Use fallback rates if no integrated rates available
				t.log.WARN.Printf("using fallback tariff with %d rates because EKZ did not return integrated rates", len(t.fallbackRates))
				t.useFallbackRates()
				once.Do(func() { close(done) })
			}
		}
		t.mux.Unlock()
	}
}

// fetchFallbackRates fetches integrated_400ST rates as fallback
func (t *Ekz) fetchFallbackRates(client *request.Helper, url string) {
	var res ekz.TariffResponse

	if err := client.GetJSON(url, &res); err != nil {
		t.log.ERROR.Printf("failed to get fallback tariff data from %s: %v", url, err)
		return
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	rates := make(api.Rates, 0, len(res.Prices))
	for _, entry := range res.Prices {
		// Find the CHF/kWh price from integrated rates (total cost including all components)
		for _, rate := range entry.Integrated {
			if rate.Unit == "CHF/kWh" {
				rates = append(rates, api.Rate{
					Start: entry.StartTimestamp,
					End:   entry.EndTimestamp,
					Value: t.totalPrice(rate.Value, entry.StartTimestamp),
				})
				break
			}
			// Skip CHF/M (monthly fixed costs) and other units
		}
	}

	if len(rates) > 0 {
		t.fallbackRates = rates
		t.log.DEBUG.Printf("updated fallback tariff with %d rates", len(rates))
	} else {
		t.log.WARN.Printf("no CHF/kWh rates found in fallback tariff response")
	}
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
