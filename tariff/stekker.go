package tariff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Map van bidding zones
var biddingZones = map[string]string{
	"BE":    "BE",
	"NL":    "NL",
	"DE-LU": "DE-LU",
	"FR":    "FR",
	"CH":    "CH",
	"SE4":   "SE4",
	"SE3":   "SE3",
	"SE1":   "SE1",
	"DK1":   "DK1",
	"DK2":   "DK2",
	"FI":    "FI",
	"NO1":   "NO1",
	"NO2":   "NO2",
	"NO3":   "NO3",
	"NO4":   "NO4",
	"NO5":   "NO5",
	"LV":    "LV",
	"LT":    "LT",
	"PL":    "PL",
	"PT":    "PT",
	"RO":    "RO",
	"RS":    "RS",
	"SI":    "SI",
	"SK":    "SK",
	"HU":    "HU",
	"AT":    "AT",
	"CZ":    "CZ",
	"HR":    "HR",
	"EE":    "EE",
}

// Stekker provider
type Stekker struct {
	uri    string
	region string // full zone name
	short  string // short code
}

// init registreert provider in registry
func init() {
	registry.Add("stekker", NewStekkerFromConfig)
}

// NewStekkerFromConfig maakt provider van config
func NewStekkerFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		Region string
		URI    string
	}{
		URI:    "https://stekker.app/epex-forecast",
		Region: "BE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	region, ok := biddingZones[cc.Region]
	if !ok {
		return nil, fmt.Errorf("unsupported zone: %s", cc.Region)
	}

	return &Stekker{
		uri:    cc.URI,
		region: region,
		short:  cc.Region,
	}, nil
}

// Type identificeert het soort tarief
func (t *Stekker) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

// Rates haalt de prijzen op van Stekker
func (t *Stekker) Rates() (api.Rates, error) {
	// Log de regio en URL
	url := fmt.Sprintf("%s?advanced_view=&region=%s&unit=MWh", t.uri, t.region)
	fmt.Println("Fetching Stekker prices for region:", t.region)
	fmt.Println("Request URL:", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`data-epex-forecast-graph-data-value="(.+?)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return nil, fmt.Errorf("no forecast data found")
	}

	raw := strings.ReplaceAll(matches[1], "&quot;", "\"")
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}

	var res api.Rates
	for _, series := range data {
		name, _ := series["name"].(string)
		if !(strings.Contains(name, "Market") || strings.Contains(name, "Forecast")) {
			continue
		}

		xs, _ := series["x"].([]interface{})
		ys, _ := series["y"].([]interface{})

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

			start = start.UTC()
			end := start.Add(time.Hour)

			res = append(res, api.Rate{
				Start: start,
				End:   end,
				Value: yt / 1000.0, // €/MWh -> €/kWh
			})
		}
	}

	fmt.Println("Fetched", len(res), "rates from Stekker")

	return res, nil
}
