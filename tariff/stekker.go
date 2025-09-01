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
	"BE":    "Belgium",
	"NL":    "Netherlands",
	"DE-LU": "Germany & Luxembourg",
	"FR":    "France",
	"CH":    "Switzerland",
	"SE4":   "Sweden SE4",
	"SE3":   "Sweden SE3",
	"SE1":   "Sweden SE1",
	"DK1":   "Denmark DK1",
	"DK2":   "Denmark DK2",
	"FI":    "Finland",
	"NO1":   "Norway NO1",
	"NO2":   "Norway NO2",
	"NO3":   "Norway NO3",
	"NO4":   "Norway NO4",
	"NO5":   "Norway NO5",
	"LV":    "Latvia",
	"LT":    "Lithuania",
	"PL":    "Poland",
	"PT":    "Portugal",
	"RO":    "Romania",
	"RS":    "Serbia",
	"SI":    "Slovenia",
	"SK":    "Slovakia",
	"HU":    "Hungary",
	"AT":    "Austria",
	"CZ":    "Czech Republic",
	"HR":    "Croatia",
	"EE":    "Estonia",
}

// Stekker provider
type Stekker struct {
	uri   string
	zone  string // full zone name
	short string // short code
}

// init registreert provider in registry
func init() {
	registry.Add("stekker", NewStekkerFromConfig)
}

// NewStekkerFromConfig maakt provider van config
func NewStekkerFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		Zone string
		URI  string
	}{
		URI:  "https://stekker.app/epex-forecast",
		Zone: "BE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	region, ok := biddingZones[cc.Zone]
	if !ok {
		return nil, fmt.Errorf("unsupported zone: %s", cc.Zone)
	}

	return &Stekker{
		uri:   cc.URI,
		zone:  region,
		short: cc.Zone,
	}, nil
}

// Type identificeert het soort tarief
func (t *Stekker) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

// Rates haalt de prijzen op van Stekker
func (t *Stekker) Rates() (api.Rates, error) {
	url := fmt.Sprintf("%s?advanced_view=&region=%s&unit=MWh", t.uri, t.zone)
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

	return res, nil
}
